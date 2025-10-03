package service

import "errors"

var (
	ErrDomainLinkNotAllowed = errors.New("domain link not in allow list")
	ErrInvalidPathFormat    = errors.New("path must contain exactly one segment")
	ErrInvalidRequestedLink = errors.New("invalid requested link")
)
