package dtos

import "encoding/json"

// ScopeOfWork defines the scope of work for different contract types
// This contains a general purpose delierables field that can hold different types of deliverables,
// depending on the contract type. The deliverables can be of type:
// - [dtos.AdvertisingDeliverable] for ADVERTISEMENT contract type
// - [dtos.AffiliateDeliverable] for AFFILIATE contract type
// - [dtos.BrandAmbassadorDeliverable] for BRAND_AMBASSADOR contract type
// - [dtos.CoProducingDeliverable] for CO_PRODUCING contract type
type ScopeOfWork struct {
	Deliverables        Deliverable `json:"deliverables"`
	GeneralRequirements []string    `json:"general_requirements,omitempty" example:"4K video, professional lighting" validate:"omitempty,max=50,dive,max=1000"`
}

// ScopeOfWorkDto is used for the conversion from the general ScopeOfWork with general Deliverables field to specific deliverable types
// This is used for marshalling into model JSONB field for database storage
type ScopeOfWorkDto struct {
	Deliverables        any      `json:"deliverables"`
	GeneralRequirements []string `json:"general_requirements,omitempty"`
}

// region: ================ Deliverable Types =================

// Deliverable is a union type that can represent any of the deliverable types
// It embeds all possible deliverable types, so it can hold fields from any of the types
// Note: This struct should NOT have a validate:"dive" tag since it's a struct, not a slice/map
type Deliverable struct {
	// AffiliateDeliverable is a subtype of AdvertisingDeliverable with some addtional fields,
	// so the fields from AdvertisingDeliverable is already embedded in the AffiliateDeliverable struct itself
	// Therefore, we don't need to embed AdvertisingDeliverable separately here
	AffiliateDeliverable
	BrandAmbassadorDeliverable
	CoProducingDeliverable
}

// AdvertisingDeliverable contains the advertised items to be created for the advertising campaign
// This is corresponding to the ADVERTISEMENT contract type
type AdvertisingDeliverable struct {
	AdvertisedItems []AdvertisedItem `json:"advertised_items,omitempty" validate:"omitempty,min=1,dive"`
}

// AffiliateDeliverable contains tracking link and platform information in addition to the fields in AdvertisingDeliverable
// This is corresponding to the AFFILIATE contract type
// Note: There can only be one TrackingLink for each Affiliate Contract
type AffiliateDeliverable struct {
	TrackingLink string   `json:"tracking_link,omitempty" example:"https://affiliate.example.com/track?ref=12345" validate:"omitempty,url,max=500"`
	Platform     []string `json:"platform,omitempty" example:"FACEBOOK,TIKTOK" validate:"omitempty,min=1,dive,oneof=FACEBOOK TIKTOK WEBSITE"`
	AdvertisingDeliverable
}

// BrandAmbassadorDeliverable contains the events that the brand ambassador needs to attend
// This is corresponding to the BRAND_AMBASSADOR contract type
type BrandAmbassadorDeliverable struct {
	Events []BrandAmbassadorEvent `json:"events,omitempty" validate:"omitempty,min=1,dive"`
}

// CoProducingDeliverable contains both products and concepts of the products and the advetisement concepts needed for that product
// This is corresponding to the CO_PRODUCING contract type
type CoProducingDeliverable struct {
	Products []CoProducingProduct `json:"products,omitempty" validate:"omitempty,min=1,dive"`
	Concepts []CoProducingConcept `json:"concepts,omitempty" validate:"omitempty,min=1,dive"`
}

// endregion

// region: ================ Sub-structures for Deliverables =================

type AdvertisedItem struct {
	ID                  *int8     `json:"id,omitempty" example:"1" validate:"omitempty,gt=0"`
	Name                string    `json:"name" example:"Product A" validate:"max=255"`
	Description         string    `json:"description,omitempty" example:"This is product A" validate:"omitempty,max=1000"`
	MaterialURL         []string  `json:"material_url" example:"https://example.com/image1.jpg,https://example.com/image2.jpg" validate:"dive,url"`
	Tagline             string    `json:"tagline" example:"Best product ever" validate:"max=255"`
	Platform            string    `json:"platform" example:"FACEBOOK" validate:"required,oneof=FACEBOOK TIKTOK WEBSITE"`
	HashTag             []string  `json:"hash_tag" example:"#bestproduct #awesome" validate:"dive,max=100"`
	CreativeNotes       string    `json:"creative_notes,omitempty" example:"Use bright colors and upbeat music" validate:"omitempty,max=1000"`
	ContentRequirements []string  `json:"content_requirements,omitempty" example:"Include product demo and customer testimonials" validate:"omitempty,dive,max=1000"`
	Metrics             []KPIGoal `json:"metrics,omitempty" validate:"dive"`
}

type BrandAmbassadorEvent struct {
	ID                  *int8     `json:"id,omitempty" example:"1" validate:"omitempty,gt=0"`
	Location            string    `json:"location" example:"Jakarta Convention Center" validate:"max=512"`
	Date                string    `json:"date" example:"2023-10-01 15:00:00" validate:"datetime=2006-01-02 15:04:05"`
	ExpectedDuration    string    `json:"expected_duration" example:"3H" validate:"omitempty,max=100"`
	Activities          string    `json:"activities" example:"Product demonstration, Q&A session" validate:"omitempty,max=1000"`
	RepresentationRules []string  `json:"representation_rules,omitempty" example:"Must wear formal attire with long leggings" validate:"omitempty,dive,max=1000"`
	KPIs                []KPIGoal `json:"kpis,omitempty" validate:"dive"`
}

type CoProducingProduct struct {
	ID          *int8     `json:"id,omitempty" example:"1" validate:"omitempty,gt=0"`
	Name        string    `json:"name" example:"Product A" validate:"max=255"`
	Description string    `json:"description,omitempty" example:"This is product A" validate:"omitempty,max=1000"`
	Materials   []string  `json:"material" example:"https://example.com/image1.jpg,https://example.com/image2.jpg" validate:"dive,url"`
	KPIs        []KPIGoal `json:"kpis,omitempty" validate:"dive"`
}

type CoProducingConcept struct {
	ProductID int8 `json:"product_id" example:"1" validate:"omitempty,gt=0"`
	AdvertisedItem
}

type KPIGoal struct {
	Metric      string `json:"metric" example:"VIEW"`
	Target      string `json:"target" example:"10000"`
	Description string `json:"description,omitempty" example:"Achieve 10,000 views on the video content"`
}

// endregion

// region: ================ Helper Conversion Functions =================

// ToAdvertisingDeliverable converts the general Deliverable struct to AdvertisingDeliverable struct through JSON marshal/unmarshal
func (d *Deliverable) ToAdvertisingDeliverable() (*AdvertisingDeliverable, error) {
	rawDeliverables, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}

	var advertisingDeliverable AdvertisingDeliverable
	err = json.Unmarshal(rawDeliverables, &advertisingDeliverable)
	if err != nil {
		return nil, err
	}

	return &advertisingDeliverable, nil
}

// ToAffiliateDeliverable converts the general Deliverable struct to AffiliateDeliverable struct through JSON marshal/unmarshal
func (d *Deliverable) ToAffiliateDeliverable() (*AffiliateDeliverable, error) {
	rawDeliverables, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}

	var affiliateDeliverable AffiliateDeliverable
	err = json.Unmarshal(rawDeliverables, &affiliateDeliverable)
	if err != nil {
		return nil, err
	}

	return &affiliateDeliverable, nil
}

// ToBrandAmbassadorDeliverable converts the general Deliverable struct to BrandAmbassadorDeliverable struct through JSON marshal/unmarshal
func (d *Deliverable) ToBrandAmbassadorDeliverable() (*BrandAmbassadorDeliverable, error) {
	rawDeliverables, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}

	var brandAmbassadorDeliverable BrandAmbassadorDeliverable
	err = json.Unmarshal(rawDeliverables, &brandAmbassadorDeliverable)
	if err != nil {
		return nil, err
	}
	return &brandAmbassadorDeliverable, nil
}

// ToCoProducingDeliverable converts the general Deliverable struct to CoProducingDeliverable struct through JSON marshal/unmarshal
func (d *Deliverable) ToCoProducingDeliverable() (*CoProducingDeliverable, error) {
	rawDeliverables, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	var coProducingDeliverable CoProducingDeliverable
	err = json.Unmarshal(rawDeliverables, &coProducingDeliverable)
	if err != nil {
		return nil, err
	}
	return &coProducingDeliverable, nil
}

// endregion
