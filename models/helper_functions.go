package models

import (
	"database/sql"
)

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

func NullFloat64OrValue(nt sql.NullFloat64) interface{} {
	if nt.Valid {
		return nt.Float64
	}
	return nil
}

func NullInt32OrValue(nt sql.NullInt32) interface{} {
	if nt.Valid {
		return nt.Int32
	}
	return nil
}

func NullInt64OrValue(nt sql.NullInt64) interface{} {
	if nt.Valid {
		return nt.Int64
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
