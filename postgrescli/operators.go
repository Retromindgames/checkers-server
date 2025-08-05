package postgrescli

import (
	"context"
	"fmt"

	"github.com/Lavizord/checkers-server/postgrescli/ent"
	"github.com/Lavizord/checkers-server/postgrescli/ent/operator"
)

func (pc *PostgresCli) GetOperators() ([]*ent.Operator, error) {
	ctx := context.Background()
	operators, err := pc.EntCli.Operator.
		Query().
		Where(operator.DeletedAtIsNil()).
		Select(operator.FieldName, operator.FieldAlias).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching operators: %v", err)
	}
	return operators, nil
}
