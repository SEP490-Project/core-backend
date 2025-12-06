package constant

type facebookVideoMetricsFields struct {
	PostVideoLikesByReactionType string
	PostVideoAvgTimeWatched      string
	PostVideoSocialActions       string
	PostVideoViewTime            string
	PostImpressionsUnique        string
	BlueReelsPlayCount           string
	FbReelsTotalPlays            string
	FbReelsReplayCount           string
	PostVideoFollowers           string
	PostVideoRetentionGraph      string
}

type facebookPostMetricsFields struct {
	PostClicks               string
	PostReactionsByTypeTotal string
	PostMediaView            string
	PostActivityByActionType string
}

type tiktokVideoMetricsFields struct {
	ViewCount    string
	LikeCount    string
	CommentCount string
	ShareCount   string
}

var (
	FacebookVideoMetrics *facebookVideoMetricsFields = &facebookVideoMetricsFields{
		PostVideoLikesByReactionType: "post_video_likes_by_reaction_type",
		PostVideoAvgTimeWatched:      "post_video_avg_time_watched",
		PostVideoSocialActions:       "post_video_social_actions",
		PostVideoViewTime:            "post_video_view_time",
		PostImpressionsUnique:        "post_impressions_unique",
		BlueReelsPlayCount:           "blue_reels_play_count",
		FbReelsTotalPlays:            "fb_reels_total_plays",
		FbReelsReplayCount:           "fb_reels_replay_count",
		PostVideoFollowers:           "post_video_followers",
		PostVideoRetentionGraph:      "post_video_retention_graph",
	}

	FacebookPostMetrics *facebookPostMetricsFields = &facebookPostMetricsFields{
		PostClicks:               "post_clicks",
		PostReactionsByTypeTotal: "post_reactions_by_type_total",
		PostMediaView:            "post_media_view",
		PostActivityByActionType: "post_activity_by_action_type",
	}

	TikTokVideoMetrics *tiktokVideoMetricsFields = &tiktokVideoMetricsFields{
		ViewCount:    "view_count",
		LikeCount:    "like_count",
		CommentCount: "comment_count",
		ShareCount:   "share_count",
	}
)
