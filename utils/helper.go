package utils

import "database/sql"

func NullStringOrValue(ns sql.NullString) interface{} {
	if ns.Valid {
		return ns.String
	}
	return nil
}

func NullTimeOrValue(nt sql.NullTime) interface{} {
	if nt.Valid {
		return nt.Time
	}
	return nil
}

func NullInt32OrValue(nt sql.NullInt32) interface{} {
	if nt.Valid {
		return nt.Int32
	}
	return nil
}

func NullInt16OrValue(nt sql.NullInt16) interface{} {
	if nt.Valid {
		return nt.Int16
	}
	return nil
}

func NullBoolOrValue(nb sql.NullBool) interface{} {
	if nb.Valid {
		return nb.Bool
	}
	return nil
}
