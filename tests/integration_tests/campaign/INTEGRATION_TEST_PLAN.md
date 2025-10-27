# Phase 7.3: Integration Test Plan for Campaign Suggestion

## Overview

This document outlines the integration test requirements for the campaign suggestion system. These tests validate end-to-end extraction flows for all 4 contract types: ADVERTISING, AFFILIATE, BRAND_AMBASSADOR, and CO_PRODUCING.

## Test Infrastructure Setup

### Required Mock Repository Methods

When implementing mocks for `GenericRepository[T]`, ensure all interface methods are implemented:

```go
// Core CRUD
GetByID(ctx, id, includes) (*T, error)
GetByCondition(ctx, filter func(*gorm.DB)*gorm.DB, includes) (*T, error) 
GetAll(ctx, filter func(*gorm.DB)*gorm.DB, includes, pageSize, pageNumber) ([]*T, int64, error)
Count(ctx, filter func(*gorm.DB)*gorm.DB) (int64, error)
Exists(ctx, filter func(*gorm.DB)*gorm.DB) (bool, error)
ExistsByID(ctx, id) (bool, error)
Add(ctx, entity) error
BulkAdd(ctx, entities, batchSize) (int64, error)
Update(ctx, entity) error
UpdateByCondition(ctx, filter func(*gorm.DB)*gorm.DB, updates) error
Delete(ctx, entity) error
DeleteByID(ctx, id) error
DB() any
```

**Important**: Filter functions must accept `func(*gorm.DB)*gorm.DB`, not `func(any)any`.

### Required Dependencies

Add to `go.mod` if missing:
```bash
go get github.com/stretchr/testify/mock@latest
```

## Test Cases by Contract Type

### 1. ADVERTISING Contract Type

**File**: `advertising_extraction_test.go`

#### Test 1.1: Multiple Schedules with Even Distribution
- **Input**: 3 advertised items, 3 payment schedules (deposit + 2 regular)
- **Expected Output**:
  - 2 milestones (deposit excluded)
  - Tasks distributed: Ceil(3/2) = 2 tasks in first milestone, 1 in second
  - Each milestone due date matches schedule date
  - Base payment = (total_cost * (1 - deposit_percent)) / num_milestones
- **Validations**:
  - Campaign name matches contract title
  - Description contains contract number and item count
  - All advertised items converted to tasks with correct type (PRODUCT)
  - Task descriptions contain: item_id, platform, materials, etc.

#### Test 1.2: Single Item, Single Schedule
- **Input**: 1 advertised item, 2 schedules (deposit + final)
- **Expected Output**:
  - 1 milestone (deposit excluded)
  - All tasks in single milestone
  - Correct payment calculation

#### Test 1.3: Error Cases
- **Contract not ACTIVE**: Should return error "only ACTIVE contracts..."
- **Empty scope of work**: Should return error "no deliverables defined..."
- **Invalid JSON**: Should return parsing error

### 2. AFFILIATE Contract Type

**File**: `affiliate_extraction_test.go`

#### Test 2.1: Monthly Payment Cycle
- **Input**: 3 advertised items, MONTHLY cycle, 3-month contract
- **Expected Output**:
  - 3 milestones (one per month)
  - All content tasks in first milestone
  - Performance tracking tasks in subsequent milestones
  - Tracking link in task descriptions

#### Test 2.2: Quarterly Payment Cycle  
- **Input**: Multiple items, QUARTERLY cycle, 1-year contract
- **Expected Output**:
  - 4 milestones (one per quarter)
  - Milestone dates match quarterly payment dates
  - Performance tracking includes tracking_link metadata

#### Test 2.3: Validation
- **Missing tracking link**: Should handle gracefully
- **Invalid payment cycle**: Should return error

### 3. BRAND_AMBASSADOR Contract Type

**File**: `brand_ambassador_extraction_test.go`

#### Test 3.1: Multiple Events with Date-Based Assignment
- **Input**: 4 events across 3 milestones
- **Expected Output**:
  - Tasks assigned to closest milestone by date
  - Event task type is CONTENT (not PRODUCT)
  - Task descriptions include: event_date, location, activities

#### Test 3.2: Single Event
- **Input**: 1 event, 1 milestone
- **Expected Output**:
  - Single task in milestone
  - Correct date extraction from event

#### Test 3.3: Edge Cases
- **Event before first milestone**: Assign to first
- **Event after last milestone**: Assign to last
- **Event exactly on milestone date**: Assign to that milestone

### 4. CO_PRODUCING Contract Type

**File**: `co_producing_extraction_test.go`

#### Test 4.1: Products and Concepts Extraction
- **Input**: 2 products, 3 concepts, MONTHLY distribution cycle
- **Expected Output**:
  - Product creation tasks (one per product)
  - Concept tasks (one per concept, linked to product_id)
  - All development tasks in first milestone
  - Performance tracking in subsequent milestones

#### Test 4.2: Profit Distribution Milestones
- **Input**: QUARTERLY distribution, 1-year contract
- **Expected Output**:
  - 4 milestones matching quarterly dates
  - Development tasks front-loaded
  - Tracking tasks include product names

#### Test 4.3: Validation
- **No products or concepts**: Should return error
- **Concepts without product_id**: Should handle gracefully

## Common Validation Patterns

### Milestone Validation
```go
// All tests should validate:
assert.Len(t, milestones, expectedCount)
assert.Equal(t, expectedDueDate, milestone.DueDate)
assert.Contains(t, milestone.Description, expectedPaymentAmount)
```

### Task Validation
```go
// All tests should validate:
assert.Equal(t, expectedName, task.Name)
assert.Equal(t, expectedType, task.Type)
assert.NotNil(t, task.Description)
assert.Contains(t, task.Description, requiredField)
```

### Distribution Algorithm
```go
// Ceil-based distribution
expectedTasksInFirst := int(math.Ceil(float64(totalTasks) / float64(totalMilestones)))
expectedTasksInRest := totalTasks - expectedTasksInFirst
```

## Test Data Structures

### Schedule Format (AdvertisingFinancialTerms)
```go
Schedules: []dtos.Schedule{
    {
        ID:       testhelpers.ToPtr(int8(1)),
        Milestone: "Deposit Payment",
        Percent:  20,
        Amount:   2000000,
        DueDate:  "2025-01-15", // Format: YYYY-MM-DD
    },
}
```

### Contract Status Values
```go
enum.ContractStatusDraft
enum.ContractStatusApproved
enum.ContractStatusActive      // Use this for valid tests
enum.ContractStatusCompleted
enum.ContractStatusInactive
enum.ContractStatusTerminated
```

### Task Types
```go
enum.TaskTypeProduct   // For advertised items
enum.TaskTypeContent   // For events, performance tracking
```

## Performance Targets

- **Small contracts** (1-10 items): <100ms
- **Medium contracts** (11-50 items): <300ms
- **Large contracts** (51-100 items): <500ms
- **XL contracts** (100+ items): <1s

## Test Execution

### Run All Integration Tests
```bash
go test ./tests/integration_tests/campaign -v
```

### Run Specific Test File
```bash
go test ./tests/integration_tests/campaign -run TestAdvertising -v
```

### Run with Coverage
```bash
go test ./tests/integration_tests/campaign -v -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## Success Criteria

- ✅ All 4 contract types have comprehensive test coverage
- ✅ Happy path scenarios pass for each type
- ✅ Error handling validated
- ✅ Milestone-payment alignment verified
- ✅ Task distribution algorithms verified
- ✅ Date handling (timezones, formats) correct
- ✅ No compilation errors
- ✅ All tests pass in <5 seconds total

## Next Steps

1. Implement mock repository with all required methods
2. Create test fixtures in `tests/testdata/campaign/`
3. Write tests following this plan
4. Add edge case tests as needed
5. Document any bugs or edge cases discovered

## Notes

- Use `testhelpers.DateOnly()` for contract dates (returns UTC time)
- Use local time for payment calculations (payment calculator returns +07:00)
- JSON field names are snake_case (e.g., `total_cost` not `totalCost`)
- Schedule uses: `Milestone`, `Percent`, `Amount`, `DueDate` (not `PaymentType`, `Percentage`, `ExpectedDate`)
- Filter functions in mocks must match signature: `func(*gorm.DB)*gorm.DB`

---

**Document Status**: DRAFT - Ready for implementation
**Last Updated**: 2025-01-30
**Phase**: 7.3 - Integration Tests
