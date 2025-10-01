package models

type CreateDurableLinkFromLongRequest struct {
	LongDurableLink string `json:"longDurableLink"`
}

type ExchangeShortLinkRequest struct {
	RequestedLink string `json:"requestedLink"`
}

type CreateDurableLinkRequest struct {
	DurableLinkInfo DurableLink `json:"durableLinkInfo"`
	Suffix          Suffix      `json:"suffix"`
}
