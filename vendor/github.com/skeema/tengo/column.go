package tengo

import (
	"fmt"
	"strings"
)

// ColumnDefault represents the default value for a column.
type ColumnDefault struct {
	Null   bool
	Quoted bool
	Value  string
}

// ColumnDefaultNull indicates a column has a default value of NULL.
var ColumnDefaultNull = ColumnDefault{Null: true}

// ColumnDefaultValue is a constructor for creating non-NULL,
// non-CURRENT_TIMESTAMP default values.
func ColumnDefaultValue(value string) ColumnDefault {
	return ColumnDefault{
		Quoted: true,
		Value:  value,
	}
}

// ColumnDefaultExpression is a constructor for creating a default value that
// represents a SQL expression, which won't be wrapped in quotes. Traditionally
// in MySQL this must be either "CURRENT_TIMESTAMP" or, if using fractional
// second precision, "CURRENT_TIMESTAMP(N)" where N is a digit.
func ColumnDefaultExpression(expression string) ColumnDefault {
	return ColumnDefault{Value: expression}
}

// Clause returns the DEFAULT clause for use in a DDL statement.
func (cd ColumnDefault) Clause() string {
	if cd.Null {
		return "DEFAULT NULL"
	} else if cd.Quoted {
		return fmt.Sprintf("DEFAULT '%s'", EscapeValueForCreateTable(cd.Value))
	} else {
		return fmt.Sprintf("DEFAULT %s", cd.Value)
	}
}

// Column represents a single column of a table.
type Column struct {
	Name          string
	TypeInDB      string
	Nullable      bool
	AutoIncrement bool
	Default       ColumnDefault
	OnUpdate      string
	CharSet       string // Only populated if textual type
	Collation     string // Only populated if textual type and differs from CharSet's default collation
	Comment       string
}

// Definition returns this column's definition clause, for use as part of a DDL
// statement. A table may optionally be supplied, which simply causes CHARACTER
// SET clause to be omitted if the table and column have the same *collation*
// (mirroring the specific display logic used by SHOW CREATE TABLE)
func (c *Column) Definition(table *Table) string {
	var charSet, collation, nullability, autoIncrement, defaultValue, onUpdate, comment string
	emitDefault := c.CanHaveDefault()
	if c.CharSet != "" && (table == nil || c.Collation != table.Collation || c.CharSet != table.CharSet) {
		// Note that we need to compare both Collation AND CharSet above, since
		// Collation of "" is used to mean default collation *for the character set*.
		charSet = fmt.Sprintf(" CHARACTER SET %s", c.CharSet)
	}
	if c.Collation != "" {
		collation = fmt.Sprintf(" COLLATE %s", c.Collation)
	}
	if !c.Nullable {
		nullability = " NOT NULL"
		if c.Default.Null {
			emitDefault = false
		}
	} else if c.TypeInDB == "timestamp" {
		// Oddly the timestamp type always displays nullability
		nullability = " NULL"
	}
	if c.AutoIncrement {
		autoIncrement = " AUTO_INCREMENT"
	}
	if emitDefault {
		defaultValue = fmt.Sprintf(" %s", c.Default.Clause())
	}
	if c.OnUpdate != "" {
		onUpdate = fmt.Sprintf(" ON UPDATE %s", c.OnUpdate)
	}
	if c.Comment != "" {
		comment = fmt.Sprintf(" COMMENT '%s'", EscapeValueForCreateTable(c.Comment))
	}
	return fmt.Sprintf("%s %s%s%s%s%s%s%s%s", EscapeIdentifier(c.Name), c.TypeInDB, charSet, collation, nullability, autoIncrement, defaultValue, onUpdate, comment)
}

// Equals returns true if two columns are identical, false otherwise.
func (c *Column) Equals(other *Column) bool {
	// shortcut if both nil pointers, or both pointing to same underlying struct
	if c == other {
		return true
	}
	// if one is nil, but we already know the two aren't equal, then we know the other is non-nil
	if c == nil || other == nil {
		return false
	}
	return *c == *other
}

// CanHaveDefault returns true if the column is allowed to have a DEFAULT clause.
func (c *Column) CanHaveDefault() bool {
	if c.AutoIncrement {
		return false
	}
	// MySQL does not permit defaults for these types
	if strings.HasSuffix(c.TypeInDB, "blob") || strings.HasSuffix(c.TypeInDB, "text") {
		return false
	}
	return true
}
