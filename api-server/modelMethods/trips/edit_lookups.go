package trips

import "context"

type DifficultyChoice struct {
	ID    int    `json:"id"`
	Level string `json:"level"`
}

type StatusChoice struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
}

type TypeChoice struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
}

type CostGroupChoice struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
}

type EditLookups struct {
	Difficulties []DifficultyChoice `json:"difficulties"`
	Statuses     []StatusChoice     `json:"statuses"`
	Types        []TypeChoice       `json:"types"`
	CostGroups   []CostGroupChoice  `json:"cost_groups"`
}

func (s Store) EditLookups(ctx context.Context) (EditLookups, error) {
	lookups := EditLookups{
		Difficulties: []DifficultyChoice{},
		Statuses:     []StatusChoice{},
		Types:        []TypeChoice{},
		CostGroups:   []CostGroupChoice{},
	}
	for _, lookup := range []struct {
		dest  any
		query string
	}{
		{&lookups.Difficulties, "SELECT id, COALESCE(level,'') AS level FROM et_trip_difficulty ORDER BY id"},
		{&lookups.Statuses, "SELECT id, COALESCE(status,'') AS status FROM et_trip_status ORDER BY id"},
		{&lookups.Types, "SELECT id, COALESCE(type,'') AS type FROM et_trip_type ORDER BY id"},
		{&lookups.CostGroups, "SELECT id, COALESCE(description,'') AS description FROM at_trip_cost_groups ORDER BY id"},
	} {
		if err := s.DB.SelectContext(ctx, lookup.dest, lookup.query); err != nil {
			return EditLookups{}, err
		}
	}
	return lookups, nil
}
