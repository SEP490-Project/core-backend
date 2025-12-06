package helper

import (
	"core-backend/internal/domain/constant"
	"core-backend/internal/domain/enum"
	"reflect"
	"strings"
)

// MapFacebookMetricsToKPIField maps Facebook metric names to corresponding KPI value types.
// For Facebook, some metrics field contains nested data that can be mapped to multiple KPI types.
// Need to pass the value parameter to determine the exact mapping if necessary.
// Currently, value can be of type float64, or map[string]float64.
func MapFacebookMetricsToKPIField(metric string, value any) map[string]float64 {
	videoMetrics := constant.FacebookVideoMetrics
	postMetrics := constant.FacebookPostMetrics
	switch metric {
	case videoMetrics.BlueReelsPlayCount, postMetrics.PostMediaView:
		// return []string{enum.KPIValueTypeReach.String()}
		if v, ok := value.(float64); ok {
			return map[string]float64{enum.KPIValueTypeReach.String(): v}
		}
		return map[string]float64{}
	case videoMetrics.PostVideoLikesByReactionType, postMetrics.PostReactionsByTypeTotal:
		if v, ok := value.(float64); ok {
			return map[string]float64{
				enum.KPIValueTypeLikes.String():      v,
				enum.KPIValueTypeEngagement.String(): v,
			}
		}
	case postMetrics.PostClicks:
		if v, ok := value.(float64); ok {
			return map[string]float64{
				enum.KPIValueTypeReach.String():      v,
				enum.KPIValueTypeEngagement.String(): v,
			}
		}
	case postMetrics.PostActivityByActionType:
		reflectedValue := reflect.ValueOf(value)
		switch reflectedValue.Kind() {
		case reflect.Map:
			keys := reflectedValue.MapKeys()
			mappedMetrics := map[string]float64{}
			if len(keys) > 0 {
				for _, key := range keys {
					if key.Kind() == reflect.String &&
						reflectedValue.MapIndex(key).Kind() == reflect.Float64 {
						mapFacebookNestedMetricsToKPIField(
							strings.ToLower(key.String()),
							reflectedValue.MapIndex(key).Float(),
							mappedMetrics)
					}
				}
			}
			return mappedMetrics
		case reflect.Float64:
			return map[string]float64{enum.KPIValueTypeEngagement.String(): reflectedValue.Float()}
		}
	}

	// Not needed: PostVideoAvgTimeWatched, PostVideoSocialActions, PostVideoViewTime, PostImpressionsUnique,
	// FbReelsTotalPlays
	return map[string]float64{}
}

func MapTikTokMetricsToKPIField(metric string, value float64) map[string]float64 {
	switch metric {
	case constant.TikTokVideoMetrics.ViewCount:
		return map[string]float64{enum.KPIValueTypeReach.String(): value}
	case constant.TikTokVideoMetrics.LikeCount:
		return map[string]float64{
			enum.KPIValueTypeLikes.String():      value,
			enum.KPIValueTypeEngagement.String(): value,
		}
	case constant.TikTokVideoMetrics.CommentCount:
		return map[string]float64{
			enum.KPIValueTypeComments.String():   value,
			enum.KPIValueTypeEngagement.String(): value,
		}
	case constant.TikTokVideoMetrics.ShareCount:
		return map[string]float64{
			enum.KPIValueTypeShares.String():     value,
			enum.KPIValueTypeEngagement.String(): value,
		}
	default:
		return map[string]float64{}
	}
}

func mapFacebookNestedMetricsToKPIField(strKey string, value float64, mappedMetrics map[string]float64) {
	if strings.Contains(strKey, "comment") {
		if existing, exists := mappedMetrics[enum.KPIValueTypeComments.String()]; exists {
			mappedMetrics[enum.KPIValueTypeComments.String()] = existing + value
		} else {
			mappedMetrics[enum.KPIValueTypeComments.String()] = value
		}
	}

	if strings.Contains(strKey, "share") {
		if existing, exists := mappedMetrics[enum.KPIValueTypeShares.String()]; exists {
			mappedMetrics[enum.KPIValueTypeShares.String()] = existing + value
		} else {
			mappedMetrics[enum.KPIValueTypeShares.String()] = value
		}
	}

	if strings.Contains(strKey, "like") ||
		strings.Contains(strKey, "love") ||
		strings.Contains(strKey, "wow") ||
		strings.Contains(strKey, "haha") ||
		strings.Contains(strKey, "sorry") ||
		strings.Contains(strKey, "anger") {
		if existing, exists := mappedMetrics[enum.KPIValueTypeLikes.String()]; exists {
			mappedMetrics[enum.KPIValueTypeLikes.String()] = existing + value
		} else {
			mappedMetrics[enum.KPIValueTypeLikes.String()] = value
		}
	}
}
