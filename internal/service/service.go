package service

import (
	"messenger/internal/domain"
	"messenger/pkg/jwt"
)

type Service struct {
	users      domain.UserRepository
	chats      domain.ChatRepository
	messages   domain.MessageRepository
	members    domain.MemberRepository
	readStates domain.ReadStateRepository
	notifier   domain.RealtimeNotifier
	presence   domain.PresenceChecker
	jwt        *jwt.Manager
}

func New(
	users domain.UserRepository,
	chats domain.ChatRepository,
	messages domain.MessageRepository,
	members domain.MemberRepository,
	readStates domain.ReadStateRepository,
	notifier domain.RealtimeNotifier,
	jwtManager *jwt.Manager,
) *Service {
	return &Service{
		users:      users,
		chats:      chats,
		messages:   messages,
		members:    members,
		readStates: readStates,
		notifier:   notifier,
		jwt:        jwtManager,
	}
}

// WithPresence attaches an in-memory presence source (WS hub). Optional; nil means everyone offline.
func (s *Service) WithPresence(presence domain.PresenceChecker) *Service {
	s.presence = presence
	return s
}
