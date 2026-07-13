import type { ReactionCounts, ReactionSummary, ReactionType } from '../types/domain'

export const REACTION_TYPES: ReactionType[] = ['like', 'dislike', 'heart']

export const REACTION_EMOJI: Record<ReactionType, string> = {
  like: '👍',
  dislike: '👎',
  heart: '❤️',
}

export const EMPTY_REACTIONS: ReactionSummary = {
  like: 0,
  dislike: 0,
  heart: 0,
  my_reaction: null,
}

export function normalizeReactions(
  reactions?: ReactionSummary | null,
): ReactionSummary {
  if (!reactions) {
    return { ...EMPTY_REACTIONS }
  }
  return {
    like: reactions.like ?? 0,
    dislike: reactions.dislike ?? 0,
    heart: reactions.heart ?? 0,
    my_reaction: reactions.my_reaction ?? null,
  }
}

export function hasAnyReaction(reactions?: ReactionSummary | null): boolean {
  const r = normalizeReactions(reactions)
  return r.like > 0 || r.dislike > 0 || r.heart > 0
}

/** Optimistic toggle/replace matching backend semantics. */
export function applyOptimisticReaction(
  current: ReactionSummary | undefined,
  reaction: ReactionType,
): ReactionSummary {
  const next = normalizeReactions(current)
  const previous = next.my_reaction

  if (previous === reaction) {
    next[reaction] = Math.max(0, next[reaction] - 1)
    next.my_reaction = null
    return next
  }

  if (previous) {
    next[previous] = Math.max(0, next[previous] - 1)
  }
  next[reaction] = next[reaction] + 1
  next.my_reaction = reaction
  return next
}

/** WS reaction_updated: refresh counts, keep local my_reaction. */
export function mergeReactionCounts(
  current: ReactionSummary | undefined,
  counts: ReactionCounts,
): ReactionSummary {
  return {
    like: counts.like,
    dislike: counts.dislike,
    heart: counts.heart,
    my_reaction: current?.my_reaction ?? null,
  }
}
