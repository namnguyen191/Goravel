package main

func doMigrate(arg2, arg3 string) error {
	dsn := getDSN()

	// run the migration commands
	switch arg2 {
	case "up":
		{
			err := grv.MigrateUp(dsn)
			if err != nil {
				return err
			}
		}
	case "down":
		{
			if arg3 == "all" {
				err := grv.MigrateDownAll(dsn)
				if err != nil {
					return err
				}
			} else {
				err := grv.Steps(-1, dsn)
				if err != nil {
					return err
				}
			}

		}
	case "reset":
		{
			err := grv.MigrateDownAll(dsn)
			if err != nil {
				return err
			}
			err = grv.MigrateUp(dsn)
			if err != nil {
				return err
			}
		}
	default:
		{
			showHelp()
		}
	}

	return nil
}
