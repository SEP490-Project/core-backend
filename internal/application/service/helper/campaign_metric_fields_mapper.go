package helper

import (
	"core-backend/internal/domain/constant"
	"core-backend/internal/domain/enum"
	"core-backend/pkg/utils"
	"reflect"
	"strings"
)

// MapFacebookMetricsToKPIField maps Facebook metric names to corresponding KPI value types.
// For Facebook, some metrics field contains nested data that can be mapped to multiple KPI types.
// Need to pass the value parameter to determine the exact mapping if necessary.
// Currently, value can be of type float64, or map[string]float64.
func MapFacebookMetricsToKPIField(metric string, value any) map[enum.KPIValueType]float64 {
	videoMetrics := constant.FacebookVideoMetrics
	postMetrics := constant.FacebookPostMetrics

	switch metric {
	// Video view metrics -> Reach (views)
	case videoMetrics.BlueReelsPlayCount, postMetrics.PostMediaView, "total_video_views":
		if v, ok := value.(float64); ok {
			return map[enum.KPIValueType]float64{enum.KPIValueTypeReach: v}
		}
		return map[enum.KPIValueType]float64{}

	// Video impressions -> Impressions
	case "total_video_impressions":
		if v, ok := value.(float64); ok {
			return map[enum.KPIValueType]float64{enum.KPIValueTypeImpressions: v}
		}
		return map[enum.KPIValueType]float64{}

	// Reactions -> Likes + Engagement
	case videoMetrics.PostVideoLikesByReactionType, postMetrics.PostReactionsByTypeTotal:
		if v, ok := value.(float64); ok {
			return map[enum.KPIValueType]float64{
				enum.KPIValueTypeLikes:      v,
				enum.KPIValueTypeEngagement: v,
			}
		}
		return map[enum.KPIValueType]float64{}

	// Post clicks -> Reach + Engagement
	case postMetrics.PostClicks:
		if v, ok := value.(float64); ok {
			return map[enum.KPIValueType]float64{
				enum.KPIValueTypeReach:      v,
				enum.KPIValueTypeEngagement: v,
			}
		}
		return map[enum.KPIValueType]float64{}

	// Activity by action type (nested map with comment, share, like breakdown)
	case postMetrics.PostActivityByActionType:
		reflectedValue := reflect.ValueOf(value)
		switch reflectedValue.Kind() {
		case reflect.Map:
			keys := reflectedValue.MapKeys()
			mappedMetrics := map[enum.KPIValueType]float64{}
			for _, key := range keys {
				if key.Kind() == reflect.String &&
					reflectedValue.MapIndex(key).Kind() == reflect.Float64 {
					mapFacebookNestedMetricsToKPIField(strings.ToLower(key.String()), reflectedValue.MapIndex(key).Float(), mappedMetrics)
				}
			}
			return mappedMetrics
		case reflect.Float64:
			return map[enum.KPIValueType]float64{enum.KPIValueTypeEngagement: reflectedValue.Float()}
		}
	}

	// Not needed: PostVideoAvgTimeWatched, PostVideoSocialActions, PostVideoViewTime, PostImpressionsUnique,
	// FbReelsTotalPlays
	return map[enum.KPIValueType]float64{}
}

func MapTikTokMetricsToKPIField(metric string, value float64) map[enum.KPIValueType]float64 {
	switch metric {
	case constant.TikTokVideoMetrics.ViewCount:
		return map[enum.KPIValueType]float64{enum.KPIValueTypeReach: value}
	case constant.TikTokVideoMetrics.LikeCount:
		return map[enum.KPIValueType]float64{
			enum.KPIValueTypeLikes:      value,
			enum.KPIValueTypeEngagement: value,
		}
	case constant.TikTokVideoMetrics.CommentCount:
		return map[enum.KPIValueType]float64{
			enum.KPIValueTypeComments:   value,
			enum.KPIValueTypeEngagement: value,
		}
	case constant.TikTokVideoMetrics.ShareCount:
		return map[enum.KPIValueType]float64{
			enum.KPIValueTypeShares:     value,
			enum.KPIValueTypeEngagement: value,
		}
	default:
		return map[enum.KPIValueType]float64{}
	}
}

func mapFacebookNestedMetricsToKPIField(strKey string, value float64, mappedMetrics map[enum.KPIValueType]float64) {
	if strings.Contains(strKey, "comment") {
		utils.AddValuesToMap(mappedMetrics, map[enum.KPIValueType]float64{
			enum.KPIValueTypeComments:   value,
			enum.KPIValueTypeEngagement: value,
		})
	}

	if strings.Contains(strKey, "share") {
		utils.AddValuesToMap(mappedMetrics, map[enum.KPIValueType]float64{
			enum.KPIValueTypeShares:     value,
			enum.KPIValueTypeEngagement: value,
		})
	}

	if strings.Contains(strKey, "like") ||
		strings.Contains(strKey, "love") ||
		strings.Contains(strKey, "wow") ||
		strings.Contains(strKey, "haha") ||
		strings.Contains(strKey, "sorry") ||
		strings.Contains(strKey, "anger") {
		utils.AddValuesToMap(mappedMetrics, map[enum.KPIValueType]float64{
			enum.KPIValueTypeLikes:      value,
			enum.KPIValueTypeEngagement: value,
		})
	}
}
