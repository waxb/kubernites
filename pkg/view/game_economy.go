package view

var DrainAges = map[string]bool{AgeEmpire: true}

func RentDrains(age string) bool {
	return DrainAges[age]
}
