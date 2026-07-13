package domain

const (
	ReactionLike    = "like"
	ReactionDislike = "dislike"
	ReactionHeart   = "heart"
)

func ValidReaction(reaction string) bool {
	switch reaction {
	case ReactionLike, ReactionDislike, ReactionHeart:
		return true
	default:
		return false
	}
}

// ReactionCounts is the aggregate without viewer-specific my_reaction.
type ReactionCounts struct {
	Like    int
	Dislike int
	Heart   int
}

// ReactionSummary is the per-message aggregate returned to a specific viewer.
type ReactionSummary struct {
	Like       int
	Dislike    int
	Heart      int
	MyReaction *string
}

type MessageWithReactions struct {
	Message
	Reactions ReactionSummary
}
