package scheduler

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/domain/model"
	"core-backend/internal/infrastructure/httpclient"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LocationSyncScheduler periodically syncs provinces/districts/wards from GHN API into local tables
// It is lightweight and controlled by config.location_sync
// - enabled: turn on/off
// - interval_minutes: period between sync runs
// - concurrency: max concurrent API calls for district/ward fetches
// It stops automatically when context is cancelled.
type locationSyncScheduler struct {
	cfg         *config.AppConfig
	db          *gorm.DB
	client      *http.Client
	enabled     bool
	syncHour    int
	concurrency int

	//Guard
	mu        sync.Mutex
	isRunning bool
}

func (s *locationSyncScheduler) StartOnce(ctx context.Context) error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		zap.L().Warn("Location sync already in progress — skipping trigger")
		return errors.New("location sync already in progress — skipping trigger")
	}
	s.isRunning = true
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			s.isRunning = false
			s.mu.Unlock()
		}()

		// Use a detached background context with timeout to ensure long-running sync isn't
		// cancelled by request-scoped contexts. Choose a reasonable upper bound for manual runs.
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		zap.L().Info("Starting location sync (manual)")
		s.safeSyncOnce(bgCtx, true)
	}()

	return nil
}

func (s *locationSyncScheduler) Start(ctx context.Context) {
	if !s.enabled {
		zap.L().Info("Location sync is disabled by config")
		return
	}

	targetHour := s.syncHour
	now := time.Now()
	firstRun := time.Date(now.Year(), now.Month(), now.Day(), targetHour, 0, 0, 0, now.Location())
	if now.After(firstRun) {
		firstRun = firstRun.Add(24 * time.Hour)
	}
	delay := time.Until(firstRun)

	zap.L().Info("Scheduler Ready! waiting for first run",
		zap.String("first_run", firstRun.Format(time.DateTime)),
		zap.Duration("delay", delay),
	)

	// schedule the first run after delay, then run every 24 hours
	go func() {
		// wait until firstRun or exit if context cancelled
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			zap.L().Info("Stopping locationSyncScheduler before first run")
			return
		case <-timer.C:
		}

		s.safeSyncOnce(ctx, false)

		// subsequent runs every 24 hours
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				zap.L().Info("Stopping locationSyncScheduler")
				return
			case <-ticker.C:
				go s.safeSyncOnce(ctx, false)
			}
		}
	}()
}

func (s *locationSyncScheduler) safeSyncOnce(ctx context.Context, manual bool) {
	mode := "automatic"
	if manual {
		mode = "manual"
	}

	zap.L().Info("Starting location sync", zap.String("mode", mode))
	start := time.Now()
	if err := s.syncOnce(ctx); err != nil {
		elapsed := time.Since(start)
		zap.L().Error("Location sync failed", zap.String("mode", mode), zap.Error(err), zap.Duration("duration", elapsed))
	} else {
		elapsed := time.Since(start)
		zap.L().Info("Location sync completed", zap.String("mode", mode), zap.Duration("duration", elapsed))
	}
}

func (s *locationSyncScheduler) syncOnce(ctx context.Context) error {
	// 1) Provinces
	provURL := s.cfg.GHN.BaseURL + "/master-data/province"
	provinces, err := httpclient.DoRequestList[responses.ProvinceResponse](ctx, s.client, s.cfg.GHN.Token, http.MethodGet, provURL, nil)
	if err != nil {
		return fmt.Errorf("fetch provinces: %w", err)
	}
	// upsert provinces
	provModels := make([]model.Province, 0, len(provinces))
	for _, p := range provinces {
		provModels = append(provModels, model.Province{
			ID:           p.ProvinceID,
			Name:         p.ProvinceName,
			CountryID:    p.CountryID,
			Code:         p.Code,
			RegionID:     p.RegionID,
			RegionCPN:    p.RegionCPN,
			IsEnable:     p.IsEnable,
			CanUpdateCOD: p.CanUpdateCOD,
			Status:       p.Status,
		})
	}
	if err := s.upsertProvinces(ctx, provModels); err != nil {
		return fmt.Errorf("upsert provinces: %w", err)
	}

	// 2) Districts per province, concurrent
	sem := make(chan struct{}, s.concurrency)
	var wg sync.WaitGroup
	var dmu sync.Mutex
	allDistricts := make([]model.District, 0, 4096)
	var dErr error
	for _, p := range provinces {
		// respect context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(provinceID int) {
			defer wg.Done()
			defer func() { <-sem }()
			url := fmt.Sprintf("%s/master-data/district?province_id=%d", s.cfg.GHN.BaseURL, provinceID)
			districts, err := httpclient.DoRequestList[responses.DistrictResponse](ctx, s.client, s.cfg.GHN.Token, http.MethodGet, url, nil)
			if err != nil {
				zap.L().Warn("Fetch districts failed", zap.Int("province_id", provinceID), zap.Error(err))
				if dErr == nil {
					dErr = err
				}
				return
			}
			local := make([]model.District, 0, len(districts))
			for _, d := range districts {
				local = append(local, model.District{
					ID:             d.DistrictID,
					ProvinceID:     d.ProvinceID,
					Name:           d.DistrictName,
					Code:           d.Code,
					Type:           d.Type,
					SupportType:    d.SupportType,
					PickType:       d.PickType,
					DeliverType:    d.DeliverType,
					GovernmentCode: d.GovernmentCode,
					IsEnable:       d.IsEnable,
					CanUpdateCOD:   d.CanUpdateCOD,
					Status:         d.Status,
				})
			}
			if err := s.upsertDistricts(ctx, local); err != nil {
				zap.L().Warn("Upsert districts failed", zap.Int("province_id", provinceID), zap.Error(err))
				if dErr == nil {
					dErr = err
				}
				return
			}
			dmu.Lock()
			allDistricts = append(allDistricts, local...)
			dmu.Unlock()
		}(p.ProvinceID)
	}
	wg.Wait()
	if dErr != nil {
		zap.L().Warn("Some district fetches/upserts failed", zap.Error(dErr))
	}

	// 3) Wards per district, concurrent
	var wmu sync.Mutex
	allWards := make([]model.Ward, 0, 8192)
	var wErr error
	wg = sync.WaitGroup{}
	for _, d := range allDistricts {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(districtID int) {
			defer wg.Done()
			defer func() { <-sem }()
			url := fmt.Sprintf("%s/master-data/ward?district_id=%d", s.cfg.GHN.BaseURL, districtID)
			wards, err := httpclient.DoRequestList[responses.WardResponse](ctx, s.client, s.cfg.GHN.Token, http.MethodGet, url, nil)
			if err != nil {
				zap.L().Warn("Fetch wards failed", zap.Int("district_id", districtID), zap.Error(err))
				if wErr == nil {
					wErr = err
				}
				return
			}
			local := make([]model.Ward, 0, len(wards))
			for _, w := range wards {
				local = append(local, model.Ward{
					Code:           w.WardCode,
					DistrictID:     w.DistrictID,
					Name:           w.WardName,
					SupportType:    w.SupportType,
					PickType:       w.PickType,
					DeliverType:    w.DeliverType,
					GovernmentCode: w.GovernmentCode,
					IsEnable:       w.IsEnable,
					CanUpdateCOD:   w.CanUpdateCOD,
					Status:         w.Status,
				})
			}
			if err := s.upsertWards(ctx, local); err != nil {
				zap.L().Warn("Upsert wards failed", zap.Int("district_id", districtID), zap.Error(err))
				if wErr == nil {
					wErr = err
				}
				return
			}
			wmu.Lock()
			allWards = append(allWards, local...)
			wmu.Unlock()
		}(d.ID)
	}
	wg.Wait()
	if wErr != nil {
		zap.L().Warn("Some ward fetches/upserts failed", zap.Error(wErr))
	}

	zap.L().Info("Location sync stats",
		zap.Int("provinces", len(provinces)),
		zap.Int("districts", len(allDistricts)),
		zap.Int("wards", len(allWards)),
	)
	return nil
}

func (s *locationSyncScheduler) upsertProvinces(ctx context.Context, items []model.Province) error {
	if len(items) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "country_id", "code", "region_id", "region_cpn", "is_enable", "can_update_cod", "status", "updated_at"}),
		}).
		Create(&items).Error
}

func (s *locationSyncScheduler) upsertDistricts(ctx context.Context, items []model.District) error {
	if len(items) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"province_id", "name", "code", "type", "support_type", "pick_type", "deliver_type", "government_code", "is_enable", "can_update_cod", "status", "updated_at"}),
		}).
		Create(&items).Error
}

func (s *locationSyncScheduler) upsertWards(ctx context.Context, items []model.Ward) error {
	if len(items) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "code"}},
			DoUpdates: clause.AssignmentColumns([]string{"district_id", "name", "support_type", "pick_type", "deliver_type", "government_code", "is_enable", "can_update_cod", "status", "updated_at"}),
		}).
		Create(&items).Error
}

func NewLocationSyncScheduler(cfg *config.AppConfig, db *gorm.DB) TaskScheduler {
	schedulerCfg := cfg.TaskScheduler.LocationSync
	cfgEnable := schedulerCfg.Enabled
	cfgSyncHour := schedulerCfg.SyncHour
	cfgConcurrency := schedulerCfg.Concurrency

	//Validate
	if cfgConcurrency <= 0 {
		cfgConcurrency = 4
	}
	if cfgSyncHour < 0 || cfgSyncHour > 23 {
		cfgSyncHour = 3
	}

	return &locationSyncScheduler{
		cfg:         cfg,
		db:          db,
		client:      &http.Client{Timeout: 60 * time.Second},
		enabled:     cfgEnable,
		syncHour:    cfgSyncHour,
		concurrency: cfgConcurrency,
	}
}
