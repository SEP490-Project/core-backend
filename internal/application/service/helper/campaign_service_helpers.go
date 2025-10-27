package helper

import (
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

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

		paymentResults, err := CalculateScheduleBasedPaymentDates(advFinancialTerms.Schedules)
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
	numberOfPeriods int,
) float64 {
	if numberOfPeriods == 0 {
		return 0
	}

	depositAmount := totalCost * (depositPercent / 100.0)
	remainingCost := totalCost - depositAmount

	return remainingCost / float64(numberOfPeriods)
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
	item dtos.AdvertisedItem,
	deadline time.Time,
) responses.SuggestedTask {
	description := buildAdvertisingTaskDescription(item)

	return responses.SuggestedTask{
		Name:        fmt.Sprintf("Create content: %s on %s", item.Name, item.Platform),
		Description: description,
		Type:        string(enum.TaskTypeContent),
		Deadline:    utils.FormatLocalTime(&deadline, ""),
	}
}

// TransformEventToTask converts a brand ambassador event to a task
func TransformEventToTask(event dtos.BrandAmbassadorEvent) responses.SuggestedTask {
	description := buildBrandAmbassadorTaskDescription(event)

	eventDate, _ := time.Parse(utils.TimeFormat, event.Date)

	return responses.SuggestedTask{
		Name:        fmt.Sprintf("Event: %s", event.Name),
		Description: description,
		Type:        string(enum.TaskTypeEvent),
		Deadline:    utils.FormatLocalTime(&eventDate, ""),
	}
}

// TransformConceptToTask converts a co-producing concept to a task
// Note: Concepts in CoProducing are stored separately with ProductID links
func TransformConceptToTask(
	concept dtos.CoProducingConcept,
	productName string,
	deadline time.Time,
) responses.SuggestedTask {
	description := buildCoProducingConceptTaskDescription(concept, productName)

	return responses.SuggestedTask{
		Name:        fmt.Sprintf("Marketing Concept: %s for %s", concept.Name, productName),
		Description: description,
		Type:        string(enum.TaskTypeContent),
		Deadline:    utils.FormatLocalTime(&deadline, ""),
	}
}

// TransformProductToCreationTask converts a co-producing product to a creation task
func TransformProductToCreationTask(
	product dtos.CoProducingProduct,
	deadline time.Time,
) responses.SuggestedTask {
	description := buildProductCreationTaskDescription(product)

	return responses.SuggestedTask{
		Name:        fmt.Sprintf("Create Product: %s", product.Name),
		Description: description,
		Type:        string(enum.TaskTypeProduct),
		Deadline:    utils.FormatLocalTime(&deadline, ""),
	}
}

// endregion

// region: ============= CO_PRODUCING Specific Extraction =============

// ExtractProductCreationTasks extracts product creation tasks from products
func ExtractProductCreationTasks(
	products []dtos.CoProducingProduct,
	deadline time.Time,
) []responses.SuggestedTask {
	tasks := make([]responses.SuggestedTask, 0, len(products))

	for _, product := range products {
		task := TransformProductToCreationTask(product, deadline)
		tasks = append(tasks, task)
	}

	return tasks
}

// ExtractConceptTasks extracts concept tasks from concepts array
// Concepts are stored separately with ProductID linking them to products
func ExtractConceptTasks(
	concepts []dtos.CoProducingConcept,
	products []dtos.CoProducingProduct,
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

		task := TransformConceptToTask(concept, productName, deadline)
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
) responses.SuggestedTask {
	dueDate, _ := time.Parse(utils.DateFormat, milestone.DueDate)

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
		Name:        taskName,
		Description: description,
		Type:        string(enum.TaskTypeContent),
		Deadline:    utils.FormatLocalTime(&dueDate, ""),
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

	if contract.DepositPercent == nil {
		return errors.New("contract deposit percent is required")
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
		milestoneDate, err := time.Parse(utils.DateFormat, milestone.DueDate)
		if err != nil {
			return fmt.Errorf("failed to parse milestone date at index %d: %w", i, err)
		}

		paymentDate := contractPayments[i].DueDate

		if !milestoneDate.Equal(paymentDate) {
			return fmt.Errorf("milestone %d due date (%s) does not match payment due date (%s)",
				i, milestoneDate.Format(utils.DateFormat), paymentDate.Format(utils.DateFormat))
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
		"kpi_goals":          item.Metrics,
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
		"kpi_goals":                concept.Metrics,
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
		milestoneDate, err := time.Parse(utils.DateFormat, milestone.DueDate)
		if err != nil {
			continue
		}

		diff := targetDate.Sub(milestoneDate)
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

// endregion
