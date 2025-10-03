package models

type ExchangeShortLinkRequest struct {
	RequestedLink string `json:"requestedLink"`
}

type CreateDurableLinkRequest struct {
	DurableLinkInfo DurableLink `json:"durableLinkInfo"`
	Suffix          Suffix      `json:"suffix"`
}
