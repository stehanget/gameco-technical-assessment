package seeds

func defaultSeeders() []Seeder {
	return []Seeder{
		NewPermissionSeeder(),
		NewRoleSeeder(),
		NewRolePermissionSeeder(),
		NewUserSeeder(),
		NewUserRoleSeeder(),
		NewCategorySeeder(),
		NewProductSeeder(),
	}
}

func productionSeeders() []Seeder {
	return []Seeder{
		NewPermissionSeeder(),
		NewRoleSeeder(),
		NewRolePermissionSeeder(),
		NewCategorySeeder(),
	}
}
