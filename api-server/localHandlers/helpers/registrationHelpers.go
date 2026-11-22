package helpers

import (
	"fmt"
	"log"

	"time"

	"github.com/jmoiron/sqlx"
)

const regDebugTag = "registrationHelpers."

// GetAgeGroupByID fetches the age group record by its ID
func GetAgeGroupByID(db *sqlx.DB, ageGroupID int64) (string, error) {
	query := `
		SELECT age_group 
		FROM et_user_age_groups 
		WHERE id = $1
	`
	var ageGroup string
	err := db.QueryRow(query, ageGroupID).Scan(&ageGroup)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return "", fmt.Errorf("age group not found")
		}
		return "", err
	}
	return ageGroup, nil
}

// ValidateAgeGroupForBirthDate validates that the given birthdate falls within the expected age range for the given age group.
// Returns true if valid, false otherwise.
// This is a basic implementation - adjust age ranges based on your business rules.
func ValidateAgeGroupForBirthDate(birthDate time.Time, ageGroupID int64, ageGroupName string) (bool, error) {
	now := time.Now()
	age := now.Year() - birthDate.Year()

	// Adjust age if birthday hasn't occurred this year
	if now.Month() < birthDate.Month() ||
		(now.Month() == birthDate.Month() && now.Day() < birthDate.Day()) {
		age--
	}

	if age < 0 {
		return false, fmt.Errorf("birthdate cannot be in the future")
	}

	// Define age ranges for common age groups (customize as needed)
	ageRanges := map[string][2]int{
		"infant":  {0, 1},
		"toddler": {1, 3},
		"child":   {3, 12},
		"youth":   {12, 18},
		"adult":   {18, 65},
		"senior":  {65, 150}, // Use high number as "no upper limit"
		"life":    {0, 150},  // Accept any age
	}

	ranges, exists := ageRanges[ageGroupName]
	if !exists {
		// If age group doesn't have defined ranges, accept it
		log.Printf("%vValidateAgeGroupForBirthDate: age group %q not in predefined ranges, accepting", regDebugTag, ageGroupName)
		return true, nil
	}

	if age >= ranges[0] && age <= ranges[1] {
		return true, nil
	}

	return false, fmt.Errorf("age %d does not fit age group %q (expected %d-%d years old)",
		age, ageGroupName, ranges[0], ranges[1])
}
