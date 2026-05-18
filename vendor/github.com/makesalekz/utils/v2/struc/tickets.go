package struc

// ------------------------ EventProvider ------------------------
type EventProvider string

const (
	EventProviderSxodim  EventProvider = "SXODIM"
	EventProviderBTikets EventProvider = "B_TIKETS"
)

func eventProviderValues() []EventProvider {
	return []EventProvider{EventProviderSxodim, EventProviderBTikets}
}

func (EventProvider) Values() (kinds []string) {
	for _, value := range eventProviderValues() {
		kinds = append(kinds, string(value))
	}
	return
}

func (m EventProvider) Value() string {
	return string(m)
}

func (m EventProvider) IsValid() bool {
	for _, value := range eventProviderValues() {
		if m == value {
			return true
		}
	}
	return false
}

// ------------------------ EventSeller ------------------------
type EventSeller string

const (
	EventSellerSxodim           EventSeller = "SXODIM"
	EventSellerAlmatyArena      EventSeller = "ALMATY_ARENA" // almaty arena
	EventSellerArenaTickets     EventSeller = "ARENA_TICKETS"
	EventSellerPalaceOfRepublic EventSeller = "PALACE_OF_REPUBLIC"
	EventSellerBTickets         EventSeller = "B_TICKETS"
)

func eventSellerValues() []EventSeller {
	return []EventSeller{
		EventSellerSxodim, EventSellerAlmatyArena, EventSellerArenaTickets, EventSellerPalaceOfRepublic, EventSellerBTickets,
	}
}

func (EventSeller) Values() (kinds []string) {
	for _, value := range eventSellerValues() {
		kinds = append(kinds, string(value))
	}
	return
}

func (m EventSeller) Value() string {
	return string(m)
}

func (m EventSeller) IsValid() bool {
	for _, value := range eventSellerValues() {
		if m == value {
			return true
		}
	}
	return false
}
