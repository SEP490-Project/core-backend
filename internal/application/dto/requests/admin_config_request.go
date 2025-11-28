package requests

type UpdateAdminConfigRequest struct {
	Value string `json:"value" validate:"required"`
}

type BulkUpdateAdminConfigRequest map[string]string
