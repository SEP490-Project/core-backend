package requests

type ProductReviewFilter struct {
	//Filters
	FromDateStr *string `form:"from_date" json:"from_date"`
	ToDateStr   *string `form:"to_date" json:"to_date"`

	RatingStarsMin *int    `form:"rating_stars_min" json:"rating_stars_min"`
	RatingStarsMax *int    `form:"rating_stars_max" json:"rating_stars_max"`
	OrderBy        *string `form:"order_by" json:"order_by"`               // created_at, rating_stars
	OrderDirection *string `form:"order_direction" json:"order_direction"` // asc, desc

	//Internal uses
	Limit int `form:"-" json:"-"`
	Page  int `form:"-" json:"-"`
}
