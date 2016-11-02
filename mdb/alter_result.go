package mdb

/*

	ALTER RESULT

*/

type AlterResult struct {
	alter        AlterStatement
	rowsAffected int
	err          error
}

func (a *AlterResult) Err() error {
	return a.err
}

func (a *AlterResult) TargetIdentifier() string {
	return a.alter.tableName
}

func (a *AlterResult) ResultCount() int {
	return a.rowsAffected
}

func (a *AlterResult) PlanDescription() string {
	return a.alter.changeStr
}

