package sharetree

// CurrentSharetree holds the current sharetree data
var CurrentSharetree []SharetreeNode

// Default root node name as per convention
const DefaultRootNodeName = "Root"

// SharetreeNode represents a node in a sharetree
type SharetreeNode struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	Type            int     `json:"type"`
	Shares          int     `json:"shares"`
	ChildNodes      string  `json:"childnodes"`
	LevelPercentage float64 `json:"levelPercentage"` // Percentage among siblings
	TotalPercentage float64 `json:"totalPercentage"` // Percentage of entire tree
}
