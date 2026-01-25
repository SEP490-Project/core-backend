package helper

import (
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/domain/constant"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// region: ============= Payment/Milestone Date Calculation =============

// GenerateMilestoneDueDatesFromFinancialTerms generates milestone due dates
// using the EXACT same logic as ContractPaymentService
func GenerateMilestoneDueDatesFromFinancialTerms(
	contract *model.Contract,
	financialTerms any,
	minimumDayBeforeDueDate int,
) ([]time.Time, error) {
	var dueDates []time.Time

	switch contract.Type {
	case enum.ContractTypeAdvertising, enum.ContractTypeAmbassador:
		advFinancialTerms, ok := financialTerms.(dtos.AdvertisingFinancialTerms)
		if !ok {
			return nil, errors.New("invalid financial terms type for ADVERTISING/AMBASSADOR contract")
		}

		paymentResults, err := CalculateScheduleBasedPaymentDates(contract.StartDate, contract.EndDate, advFinancialTerms.Schedules)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate schedule-based payment dates: %w", err)
		}

		for _, result := range paymentResults {
			dueDates = append(dueDates, result.DueDate)
		}

	case enum.ContractTypeAffiliate:
		affFinancialTerms, ok := financialTerms.(dtos.AffiliateFinancialTerms)
		if !ok {
			return nil, errors.New("invalid financial terms type for AFFILIATE contract")
		}

		paymentCycle := enum.PaymentCycle(affFinancialTerms.PaymentCycle)
		if !paymentCycle.IsValid() {
			return nil, fmt.Errorf("invalid payment cycle: %s", affFinancialTerms.PaymentCycle)
		}

		paymentResults, err := CalculatePaymentDatesForCycle(
			paymentCycle,
			contract.StartDate,
			contract.EndDate,
			affFinancialTerms.PaymentDate,
			minimumDayBeforeDueDate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate affiliate payment dates: %w", err)
		}

		for _, result := range paymentResults {
			dueDates = append(dueDates, result.DueDate)
		}

	case enum.ContractTypeCoProduce:
		coProducingTerms, ok := financialTerms.(dtos.CoProducingFinancialTerms)
		if !ok {
			return nil, errors.New("invalid financial terms type for CO_PRODUCING contract")
		}

		paymentCycle := enum.PaymentCycle(coProducingTerms.ProfitDistributionCycle)
		if !paymentCycle.IsValid() {
			return nil, fmt.Errorf("invalid profit distribution cycle: %s", coProducingTerms.ProfitDistributionCycle)
		}

		paymentResults, err := CalculatePaymentDatesForCycle(
			paymentCycle,
			contract.StartDate,
			contract.EndDate,
			coProducingTerms.ProfitDistributionDate,
			minimumDayBeforeDueDate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate co-producing payment dates: %w", err)
		}

		for _, result := range paymentResults {
			dueDates = append(dueDates, result.DueDate)
		}

	default:
		return nil, fmt.Errorf("unsupported contract type: %s", contract.Type)
	}

	return dueDates, nil
}

// CalculateBasePaymentPerPeriod calculates the base payment amount per period
// Formula: (TotalCost - Deposit) / NumberOfPeriods
// TotalCost must be extracted from FinancialTerms based on contract type
func CalculateBasePaymentPerPeriod(
	totalCost float64,
	depositPercent float64,
	actualDepositAmount *int,
	numberOfPeriods int,
) (baseAmount, amountPercent float64) {
	if numberOfPeriods == 0 {
		return 0, 0
	}

	// depositAmount := totalCost * (depositPercent / 100.0)
	var depositAmount float64
	if actualDepositAmount != nil {
		depositAmount = float64(*actualDepositAmount)
	} else {
		depositAmount = totalCost * (depositPercent / 100.0)
	}
	remainingCost := totalCost - depositAmount

	return remainingCost / float64(numberOfPeriods), (remainingCost / totalCost) * 100.0 / float64(numberOfPeriods)
}

// ExtractTotalCostFromFinancialTerms extracts TotalCost from FinancialTerms JSONB
func ExtractTotalCostFromFinancialTerms(contract *model.Contract) (float64, error) {
	switch contract.Type {
	case enum.ContractTypeAdvertising, enum.ContractTypeAmbassador:
		var financialTerms dtos.AdvertisingFinancialTerms
		if err := json.Unmarshal(contract.FinancialTerms, &financialTerms); err != nil {
			return 0, fmt.Errorf("failed to unmarshal advertising financial terms: %w", err)
		}
		return float64(financialTerms.TotalCost), nil

	case enum.ContractTypeAffiliate:
		var financialTerms dtos.AffiliateFinancialTerms
		if err := json.Unmarshal(contract.FinancialTerms, &financialTerms); err != nil {
			return 0, fmt.Errorf("failed to unmarshal affiliate financial terms: %w", err)
		}
		return float64(financialTerms.TotalCost), nil

	case enum.ContractTypeCoProduce:
		var financialTerms dtos.CoProducingFinancialTerms
		if err := json.Unmarshal(contract.FinancialTerms, &financialTerms); err != nil {
			return 0, fmt.Errorf("failed to unmarshal co-producing financial terms: %w", err)
		}
		return float64(financialTerms.TotalCost), nil

	default:
		return 0, fmt.Errorf("unsupported contract type: %s", contract.Type)
	}
}

// endregion

// region: ============= Task Transformation =============

// TransformAdvertisedItemToTask converts an advertised item to a suggested task
func TransformAdvertisedItemToTask(
	item dtos.AdvertisedItem, contractID uuid.UUID, deadline time.Time,
) responses.SuggestedTask {
	description := buildAdvertisingTaskDescription(item)
	var scopeOfWorkID *string
	if item.ID != nil {
		scopeOfWorkID = utils.PtrOrNil(fmt.Sprintf("%s|%s|%d", contractID.String(), constant.ScopeOfWorkIDTypeAdvertise, *item.ID))
	}

	return responses.SuggestedTask{
		Name:              fmt.Sprintf("Create content: %s on %s", item.Name, item.Platform),
		Description:       description,
		Type:              enum.TaskTypeContent,
		Deadline:          deadline,
		ScopeOfWorkItemID: scopeOfWorkID,
	}
}

// TransformEventToTask converts a brand ambassador event to a task
func TransformEventToTask(event dtos.BrandAmbassadorEvent, contractID uuid.UUID) responses.SuggestedTask {
	description := buildBrandAmbassadorTaskDescription(event)
	eventDate, _ := time.Parse(utils.TimeFormat, event.Date)
	var scopeOfWorkID *string
	if event.ID != nil {
		scopeOfWorkID = utils.PtrOrNil(fmt.Sprintf("%s|%s|%d", contractID.String(), constant.ScopeOfWorkIDTypeEvent, *event.ID))
	}

	return responses.SuggestedTask{
		Name:              fmt.Sprintf("Event: %s", event.Name),
		Description:       description,
		Type:              enum.TaskTypeEvent,
		Deadline:          eventDate,
		ScopeOfWorkItemID: scopeOfWorkID,
	}
}

// TransformConceptToTask converts a co-producing concept to a task
// Note: Concepts in CoProducing are stored separately with ProductID links
func TransformConceptToTask(
	concept dtos.CoProducingConcept,
	contractID uuid.UUID,
	productName string,
	deadline time.Time,
) responses.SuggestedTask {
	description := buildCoProducingConceptTaskDescription(concept, productName)
	var scopeOfWorkID *string
	if concept.ID != nil {
		scopeOfWorkID = utils.PtrOrNil(fmt.Sprintf("%s|%s|%d", contractID.String(), constant.ScopeOfWorkIDTypeConcept, *concept.ID))
	}

	return responses.SuggestedTask{
		Name:              fmt.Sprintf("Marketing Concept: %s for %s", concept.Name, productName),
		Description:       description,
		Type:              enum.TaskTypeContent,
		Deadline:          deadline,
		ScopeOfWorkItemID: scopeOfWorkID,
	}
}

// TransformProductToCreationTask converts a co-producing product to a creation task
func TransformProductToCreationTask(
	product dtos.CoProducingProduct,
	contractID uuid.UUID,
	deadline time.Time,
) responses.SuggestedTask {
	description := buildProductCreationTaskDescription(product)
	var scopeOfWorkID *string
	if product.ID != nil {
		scopeOfWorkID = utils.PtrOrNil(fmt.Sprintf("%s|%s|%d", contractID.String(), constant.ScopeOfWorkIDTypeProduct, *product.ID))
	}

	return responses.SuggestedTask{
		Name:              fmt.Sprintf("Create Product: %s", product.Name),
		Description:       description,
		Type:              enum.TaskTypeProduct,
		Deadline:          deadline,
		ScopeOfWorkItemID: scopeOfWorkID,
	}
}

// endregion

// region: ============= CO_PRODUCING Specific Extraction =============

// ExtractProductCreationTasks extracts product creation tasks from products
func ExtractProductCreationTasks(
	products []dtos.CoProducingProduct,
	contract *model.Contract,
	deadline time.Time,
) []responses.SuggestedTask {
	tasks := make([]responses.SuggestedTask, 0, len(products))

	for _, product := range products {
		task := TransformProductToCreationTask(product, contract.ID, deadline)
		tasks = append(tasks, task)
	}

	return tasks
}

// ExtractConceptTasks extracts concept tasks from concepts array
// Concepts are stored separately with ProductID linking them to products
func ExtractConceptTasks(
	concepts []dtos.CoProducingConcept,
	products []dtos.CoProducingProduct,
	contract *model.Contract,
	deadline time.Time,
) []responses.SuggestedTask {
	// Create product lookup map
	productMap := make(map[int8]string)
	for _, product := range products {
		if product.ID != nil {
			productMap[*product.ID] = product.Name
		}
	}

	var tasks []responses.SuggestedTask
	for _, concept := range concepts {
		productName := productMap[concept.ProductID]
		if productName == "" {
			productName = "Unknown Product"
		}

		task := TransformConceptToTask(concept, contract.ID, productName, deadline)
		tasks = append(tasks, task)
	}

	return tasks
}

// endregion

// region: ============= Task Assignment =============

// DistributeTasksEvenly distributes tasks evenly across milestones
func DistributeTasksEvenly(
	tasks []responses.SuggestedTask,
	milestones []responses.SuggestedMilestone,
) []responses.SuggestedMilestone {
	if len(milestones) == 0 {
		return milestones
	}

	if len(milestones) == 1 {
		milestones[0].Tasks = tasks
		return milestones
	}

	tasksPerMilestone := int(math.Ceil(float64(len(tasks)) / float64(len(milestones))))
	taskIndex := 0

	for i := range milestones {
		end := min(taskIndex+tasksPerMilestone, len(tasks))

		milestones[i].Tasks = tasks[taskIndex:end]
		taskIndex = end

		if taskIndex >= len(tasks) {
			break
		}
	}

	return milestones
}

// AssignTasksByDate assigns tasks to milestones based on closest due date
func AssignTasksByDate(
	tasks []responses.SuggestedTask,
	milestones []responses.SuggestedMilestone,
) []responses.SuggestedMilestone {
	if len(milestones) == 0 {
		return milestones
	}

	for _, task := range tasks {
		taskDate, err := ExtractDateFromTask(task)
		if err != nil {
			zap.L().Warn("Failed to extract date from task, assigning to first milestone",
				zap.String("task_name", task.Name),
				zap.Error(err))
			milestones[0].Tasks = append(milestones[0].Tasks, task)
			continue
		}

		closestIndex := FindClosestMilestoneIndex(taskDate, milestones)
		milestones[closestIndex].Tasks = append(milestones[closestIndex].Tasks, task)
	}

	return milestones
}

// AssignAffiliateTasksToMilestones assigns affiliate tasks: all content to first, tracking to rest
func AssignAffiliateTasksToMilestones(
	contentTasks []responses.SuggestedTask,
	milestones []responses.SuggestedMilestone,
	trackingLink string,
	updatedByID *uuid.UUID,
) []responses.SuggestedMilestone {
	if len(milestones) == 0 {
		return milestones
	}

	// All content tasks → first milestone
	milestones[0].Tasks = contentTasks

	// Generate tracking tasks for remaining milestones
	for i := 1; i < len(milestones); i++ {
		trackingTask := GeneratePerformanceTrackingTask(
			milestones[i],
			"AFFILIATE",
			trackingLink,
			updatedByID,
		)
		milestones[i].Tasks = []responses.SuggestedTask{trackingTask}
	}

	return milestones
}

// AssignCoProducingTasksToMilestones assigns co-producing tasks: all development to first, tracking to rest
func AssignCoProducingTasksToMilestones(
	allDevelopmentTasks []responses.SuggestedTask,
	milestones []responses.SuggestedMilestone,
	productNames []string,
	updatedByID *uuid.UUID,
) []responses.SuggestedMilestone {
	if len(milestones) == 0 {
		return milestones
	}

	// All development tasks (product + concept) → first milestone
	milestones[0].Tasks = allDevelopmentTasks

	// Generate tracking tasks for remaining milestones
	for i := 1; i < len(milestones); i++ {
		trackingTask := GeneratePerformanceTrackingTask(
			milestones[i],
			"CO_PRODUCING",
			fmt.Sprintf("Products: %v", productNames),
			updatedByID,
		)
		milestones[i].Tasks = []responses.SuggestedTask{trackingTask}
	}

	return milestones
}

// endregion

// region: ============= Performance Tracking =============

// GeneratePerformanceTrackingTask creates a performance review task
func GeneratePerformanceTrackingTask(
	milestone responses.SuggestedMilestone,
	contractType string,
	metadata string,
	updatedByID *uuid.UUID,
) responses.SuggestedTask {
	dueDate := milestone.DueDate

	var taskName string
	var description map[string]any

	if contractType == "AFFILIATE" {
		taskName = fmt.Sprintf("Review CTR Metrics for period ending %s", milestone.DueDate)
		description = map[string]any{
			"task_type":         "PERFORMANCE_REVIEW",
			"contract_type":     "AFFILIATE",
			"tracking_link":     metadata,
			"metrics_to_review": []string{"Total Clicks", "CTR %", "Commission Earned"},
			"instructions":      "Review affiliate performance metrics before milestone completion to calculate performance-based payment",
			"period_end":        milestone.DueDate,
		}
	} else {
		taskName = fmt.Sprintf("Review Sales Data for period ending %s", milestone.DueDate)
		description = map[string]any{
			"task_type":         "PERFORMANCE_REVIEW",
			"contract_type":     "CO_PRODUCING",
			"products":          metadata,
			"metrics_to_review": []string{"Units Sold", "Revenue Generated", "Profit Share Calculated"},
			"instructions":      "Review product sales data before milestone completion to calculate profit-based payment",
			"period_end":        milestone.DueDate,
		}
	}

	return responses.SuggestedTask{
		Name:         taskName,
		Description:  description,
		Type:         enum.TaskTypeOther,
		Deadline:     dueDate,
		AssignedToID: updatedByID,
	}
}

//endregion

// region: ============= Validation =============

// ValidateContractForSuggestion validates contract has required fields
func ValidateContractForSuggestion(contract *model.Contract) error {
	if contract == nil {
		return errors.New("contract is nil")
	}

	if contract.StartDate.IsZero() {
		return errors.New("contract start date is required")
	}

	if contract.EndDate.IsZero() {
		return errors.New("contract end date is required")
	}

	if len(contract.ScopeOfWork) == 0 {
		return errors.New("contract scope of work is required")
	}

	if len(contract.FinancialTerms) == 0 {
		return errors.New("contract financial terms is required")
	}

	// Validate TotalCost exists in FinancialTerms
	totalCost, err := ExtractTotalCostFromFinancialTerms(contract)
	if err != nil {
		return fmt.Errorf("failed to extract total cost: %w", err)
	}

	if totalCost <= 0 {
		return errors.New("contract total cost must be greater than zero")
	}

	if contract.DepositAmount == nil && contract.DepositPercent == nil {
		return errors.New("contract deposit amount or percent is required")
	}

	return nil
}

// ValidateMilestonePaymentAlignment validates that milestones align with contract payments
func ValidateMilestonePaymentAlignment(
	milestones []responses.SuggestedMilestone,
	contractPayments []*model.ContractPayment,
) error {
	if len(milestones) != len(contractPayments) {
		return fmt.Errorf("milestone count (%d) does not match contract payment count (%d)",
			len(milestones), len(contractPayments))
	}

	for i, milestone := range milestones {
		paymentDate := contractPayments[i].DueDate

		if !milestone.DueDate.Equal(paymentDate) {
			return fmt.Errorf("milestone %d due date (%s) does not match payment due date (%s)",
				i, milestone.DueDate.Format(utils.DateFormat), paymentDate.Format(utils.DateFormat))
		}
	}

	return nil
}

// endregion

// region: ============= Description Generation =============

// buildAdvertisingTaskDescription builds JSON description for advertising task
func buildAdvertisingTaskDescription(item dtos.AdvertisedItem) map[string]any {
	return map[string]any{
		"advertised_item_id": item.ID,
		"product_name":       item.Name,
		"platform":           item.Platform,
		"hashtags":           item.HashTag,
		"material_urls":      item.MaterialURL,
		"kpi_goals":          item.KPIs,
		"tagline":            item.Tagline,
		"creative_notes":     item.CreativeNotes,
	}
}

// buildBrandAmbassadorTaskDescription builds JSON description for event task
func buildBrandAmbassadorTaskDescription(event dtos.BrandAmbassadorEvent) map[string]any {
	return map[string]any{
		"event_id":             event.ID,
		"event_name":           event.Name,
		"event_date":           event.Date,
		"location":             event.Location,
		"activities":           event.Activities,
		"kpi_goals":            event.KPIs,
		"representation_rules": event.RepresentationRules,
		"event_duration":       event.ExpectedDuration,
	}
}

// buildCoProducingConceptTaskDescription builds JSON description for concept task
func buildCoProducingConceptTaskDescription(
	concept dtos.CoProducingConcept,
	productName string,
) map[string]any {
	return map[string]any{
		"concept_id":               concept.ID,
		"concept_name":             concept.Name,
		"concept_description":      concept.Description,
		"is_product_creation_task": false,
		"related_product_id":       concept.ProductID,
		"related_product_name":     productName,
		"materials":                concept.MaterialURL,
		"kpi_goals":                concept.KPIs,
		"platform":                 concept.Platform,
		"hashtags":                 concept.HashTag,
	}
}

// buildProductCreationTaskDescription builds JSON description for product creation task
func buildProductCreationTaskDescription(product dtos.CoProducingProduct) map[string]any {
	return map[string]any{
		"product_id":               product.ID,
		"product_name":             product.Name,
		"is_product_creation_task": true,
		"product_description":      product.Description,
		"kpi_goals":                product.KPIs,
		"materials":                product.Materials,
		"subtasks": []string{
			"Define base product specifications",
			"Create product variants (colors, sizes)",
			"Write product story and description",
			"Design product packaging",
		},
	}
}

// endregion

// region: ============= Utility =============

// ExtractDateFromTask extracts date from task description for date-based assignment
func ExtractDateFromTask(task responses.SuggestedTask) (time.Time, error) {
	descMap := task.Description

	// Try event_date for brand ambassador tasks
	if eventDateStr, ok := descMap["event_date"].(string); ok {
		return time.Parse(utils.TimeFormat, eventDateStr)
	}

	return time.Time{}, errors.New("no date found in task description")
}

// FindClosestMilestoneIndex finds the index of the milestone with closest due date to target
func FindClosestMilestoneIndex(targetDate time.Time, milestones []responses.SuggestedMilestone) int {
	if len(milestones) == 0 {
		return 0
	}

	closestIndex := 0
	minDiff := time.Hour * 24 * 365 * 100 // 100 years

	for i, milestone := range milestones {
		diff := targetDate.Sub(milestone.DueDate)
		if diff < 0 {
			diff = -diff
		}

		if diff < minDiff {
			minDiff = diff
			closestIndex = i
		}
	}

	return closestIndex
}

// ExtractProductNames extracts product names from products array
func ExtractProductNames(products []dtos.CoProducingProduct) []string {
	names := make([]string, len(products))
	for i, product := range products {
		names[i] = product.Name
	}
	return names
}

// MapTasksToScopeOfWork maps tasks to SOW items based on name matching using a scoring system
func MapTasksToScopeOfWork(contract *model.Contract, milestones []*model.Milestone) error {
	var sow dtos.ScopeOfWork
	if err := json.Unmarshal(contract.ScopeOfWork, &sow); err != nil {
		return err
	}

	// 1. Collect all SOW items as candidates
	type sowItemCandidate struct {
		ID         *int8
		Name       string
		Type       string // "AdvertisedItem", "Event", "Product", "Concept"
		Platform   string
		TaskIDs    *[]uuid.UUID
		ProductIDs *[]uuid.UUID
		ContentIDs *[]uuid.UUID
	}
	var candidates []sowItemCandidate

	addCandidate := func(id *int8, name, itemType, platform string, tIDs, pIDs, cIDs *[]uuid.UUID) {
		if id != nil {
			candidates = append(candidates, sowItemCandidate{id, name, itemType, platform, tIDs, pIDs, cIDs})
		}
	}

	switch contract.Type {
	case enum.ContractTypeAffiliate:
		for i := range sow.Deliverables.AdvertisedItems {
			item := &sow.Deliverables.AdvertisedItems[i]
			addCandidate(item.ID, item.Name, constant.ScopeOfWorkIDTypeAffiliate.String(), item.Platform, &item.TaskIDs, nil, &item.ContentIDs)
		}

	case enum.ContractTypeAdvertising:
		for i := range sow.Deliverables.AdvertisedItems {
			item := &sow.Deliverables.AdvertisedItems[i]
			addCandidate(item.ID, item.Name, constant.ScopeOfWorkIDTypeAdvertise.String(), item.Platform, &item.TaskIDs, nil, &item.ContentIDs)
		}

	case enum.ContractTypeAmbassador:
		for i := range sow.Deliverables.Events {
			item := &sow.Deliverables.Events[i]
			addCandidate(item.ID, item.Name, constant.ScopeOfWorkIDTypeEvent.String(), "", &item.TaskIDs, nil, nil)
		}

	case enum.ContractTypeCoProduce:
		for i := range sow.Deliverables.Products {
			item := &sow.Deliverables.Products[i]
			addCandidate(item.ID, item.Name, constant.ScopeOfWorkIDTypeProduct.String(), "", &item.TaskIDs, &item.ProductIDs, nil)
		}
		for i := range sow.Deliverables.Concepts {
			item := &sow.Deliverables.Concepts[i]
			addCandidate(item.ID, item.Name, constant.ScopeOfWorkIDTypeConcept.String(), item.Platform, &item.TaskIDs, &item.ProductIDs, nil)
		}
	}

	// 2. Iterate through all tasks and find the best SOW item match
	for _, m := range milestones {
		for _, t := range m.Tasks {
			// If already assigned manually, ensure consistency and skip matching
			if t.ScopeOfWorkItemID != nil {
				idStr := strings.Split(*t.ScopeOfWorkItemID, "|")[2] // Extract SOW item ID part
				for _, cand := range candidates {
					if fmt.Sprintf("%d", *cand.ID) == idStr {
						// Ensure ID is in cand.TaskIDs
						found := slices.Contains(*cand.TaskIDs, t.ID)
						if !found {
							*cand.TaskIDs = append(*cand.TaskIDs, t.ID)
						}
						// Update Product/Content IDs for manually assigned tasks
						updateRelatedIDs(t, cand.ProductIDs, cand.ContentIDs)
					}
				}
				continue
			}

			// Single Item Rule: If there is only one candidate, assign all unassigned tasks to it
			if len(candidates) == 1 {
				cand := &candidates[0]
				idStr := fmt.Sprintf("%s|%s|%d", contract.ID.String(), cand.Type, *cand.ID)
				t.ScopeOfWorkItemID = &idStr
				*cand.TaskIDs = append(*cand.TaskIDs, t.ID)
				updateRelatedIDs(t, cand.ProductIDs, cand.ContentIDs)
				continue
			}

			// Find best match among candidates
			var bestMatch *sowItemCandidate
			bestScore := 0

			for i := range candidates {
				cand := &candidates[i]
				score := calculateMatchScore(t, cand.Name, cand.Type, cand.Platform)
				if score > bestScore {
					bestScore = score
					bestMatch = cand
				}
			}

			// Threshold for assignment (e.g., 40 points)
			if bestMatch != nil && bestScore >= 40 {
				idStr := fmt.Sprintf("%s|%s|%d", contract.ID.String(), bestMatch.Type, *bestMatch.ID)
				t.ScopeOfWorkItemID = &idStr

				// Add to TaskIDs
				*bestMatch.TaskIDs = append(*bestMatch.TaskIDs, t.ID)

				// Update Product/Content IDs
				updateRelatedIDs(t, bestMatch.ProductIDs, bestMatch.ContentIDs)
			}
		}
	}

	// Marshal back
	newSOW, err := json.Marshal(sow)
	if err != nil {
		return err
	}
	contract.ScopeOfWork = newSOW
	return nil
}

// updateRelatedIDs updates ProductIDs and ContentIDs lists if the task has them
func updateRelatedIDs(t *model.Task, productIDs *[]uuid.UUID, contentIDs *[]uuid.UUID) {
	// Update ProductIDs
	if productIDs != nil && t.Product != nil {
		foundP := slices.Contains(*productIDs, t.Product.ID)
		if !foundP {
			*productIDs = append(*productIDs, t.Product.ID)
		}
	}

	// Update ContentIDs
	if contentIDs != nil && len(t.Contents) > 0 {
		for _, content := range t.Contents {
			foundC := slices.Contains(*contentIDs, content.ID)
			if !foundC {
				*contentIDs = append(*contentIDs, content.ID)
			}
		}
	}
}

// calculateMatchScore returns a score from 0-100 indicating how well the task name matches the item name
func calculateMatchScore(task *model.Task, itemName, itemType, platform string) int {
	taskName := task.Name
	tName := strings.ToLower(strings.TrimSpace(taskName))
	iName := strings.ToLower(strings.TrimSpace(itemName))

	score := 0

	if tName == "" || iName == "" {
		return 0
	}

	// 1. Exact Match
	if tName == iName {
		score = 100
	} else if strings.Contains(tName, iName) {
		// 2. Containment (Task Name contains Item Name)
		score = 80
	} else if strings.Contains(iName, tName) {
		// 3. Reverse Containment (Item Name contains Task Name)
		score = 60
	} else {
		// 4. Word Overlap
		tWords := strings.Fields(tName)
		iWords := strings.Fields(iName)
		if len(tWords) > 0 && len(iWords) > 0 {
			matches := 0
			for _, tw := range tWords {
				if slices.Contains(iWords, tw) {
					matches++
				}
			}

			maxLen := max(len(iWords), len(tWords))

			overlapScore := min(int((float64(matches)/float64(maxLen))*100), 50)
			score = overlapScore
		}
	}

	// 5. Heuristics Bonus
	// Type Matching
	if itemType == "Event" && task.Type == enum.TaskTypeEvent {
		score += 20
	}
	if itemType == "Product" && task.Type == enum.TaskTypeProduct {
		score += 20
	}
	if (itemType == "AdvertisedItem" || itemType == "Concept") && task.Type == enum.TaskTypeContent {
		score += 10
	}

	// Platform Matching
	if platform != "" {
		if strings.Contains(tName, strings.ToLower(platform)) {
			score += 20
		}
	}

	return score
}

// endregion
