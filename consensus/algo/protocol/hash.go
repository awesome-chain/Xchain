package protocol

// HashID is a domain separation prefix for an object type that might be hashed
// This ensures, for example, the hash of a transaction will never collide with the hash of a vote
type HashID string

// Hash IDs for specific object types, in lexicographic order to avoid dups.
const (
	AuctionBid        HashID = "aB"
	AuctionDeposit    HashID = "aD"
	AuctionOutcomes   HashID = "aO"
	AuctionParams     HashID = "aP"
	AuctionSettlement HashID = "aS"

	AgreementSelector HashID = "AS"
	BlockHeader       HashID = "BH"
	BalanceRecord     HashID = "BR"
	Credential        HashID = "CR"
	Genesis           HashID = "GE"
	Message           HashID = "MX"
	NetPrioResponse   HashID = "NPR"
	OneTimeSigKey1    HashID = "OT1"
	OneTimeSigKey2    HashID = "OT2"
	PaysetFlat        HashID = "PF"
	Payload           HashID = "PL"
	ProposerSeed      HashID = "PS"
	Seed              HashID = "SD"
	TestHashable      HashID = "TE"
	Transaction       HashID = "TX"
	Vote              HashID = "VO"
)
