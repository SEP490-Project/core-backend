package service

import (
	"bytes"
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/domain/model"
	"encoding/json"
	"fmt"
	"io"
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
// It stops automatically when context is canceled.
type LocationSyncScheduler struct {
	cfg         *config.AppConfig
	db          *gorm.DB
	client      *http.Client
	interval    time.Duration
	concurrency int
}

func NewLocationSyncScheduler(cfg *config.AppConfig, db *gorm.DB) *LocationSyncScheduler {
	interval := time.Duration(cfg.LocationSync.IntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	concurrency := cfg.LocationSync.Concurrency
	if concurrency <= 0 {
		concurrency = 4
	}
	return &LocationSyncScheduler{
		cfg:         cfg,
		db:          db,
		client:      &http.Client{Timeout: 30 * time.Second},
		interval:    interval,
		concurrency: concurrency,
	}
}

func (s *LocationSyncScheduler) Start(ctx context.Context) {
	if !s.cfg.LocationSync.Enabled {
		zap.L().Info("Location sync is disabled by config")
		return
	}

	zap.L().Info("Starting LocationSyncScheduler",
		zap.Duration("interval", s.interval),
		zap.Int("concurrency", s.concurrency),
	)

	// initial run
	go s.safeSyncOnce(ctx)

	ticker := time.NewTicker(s.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				zap.L().Info("Stopping LocationSyncScheduler")
				return
			case <-ticker.C:
				go s.safeSyncOnce(ctx)
			}
		}
	}()
}

func (s *LocationSyncScheduler) safeSyncOnce(ctx context.Context) {
	if err := s.syncOnce(ctx); err != nil {
		zap.L().Error("Location sync failed", zap.Error(err))
	} else {
		zap.L().Info("Location sync completed")
	}
}

func (s *LocationSyncScheduler) syncOnce(ctx context.Context) error {
	// 1) Provinces
	provURL := s.cfg.GHN.BaseURL + "/province"
	provinces, err := s.doRequest[responses.ProvinceResponse](ctx, http.MethodGet, provURL, nil)
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
			url := fmt.Sprintf("%s/district?province_id=%d", s.cfg.GHN.BaseURL, provinceID)
			districts, err := s.doRequest[responses.DistrictResponse](ctx, http.MethodGet, url, nil)
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
			url := fmt.Sprintf("%s/ward?district_id=%d", s.cfg.GHN.BaseURL, districtID)
			wards, err := doRequest()[responses.WardResponse](ctx, http.MethodGet, url, nil)
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

func (s *LocationSyncScheduler) upsertProvinces(ctx context.Context, items []model.Province) error {
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

func (s *LocationSyncScheduler) upsertDistricts(ctx context.Context, items []model.District) error {
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

func (s *LocationSyncScheduler) upsertWards(ctx context.Context, items []model.Ward) error {
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

func doRequest[T any](ctx context.Context, client *http.Client, token string, method, url string, body any) ([]T, error) {
	var buf io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		buf = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, buf)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Token", token)
	if body != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		zap.L().Warn("Non-200 from GHN", zap.Int("status", resp.StatusCode), zap.String("body", string(b)))
	}

	var result responses.GHNAPIResponse[T]
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}
