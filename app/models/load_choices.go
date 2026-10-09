package models

// LoadChoice is shared by load forms and registration preferences.
type LoadChoice struct {
	Value string
	Label string
}

func CargoChoices() []LoadChoice {
	return []LoadChoice{{"FTL", "FTL · Full Truckload"}, {"LTL", "LTL · Partial"}, {"parcel", "Paquetería"}, {"bulk", "Granel seco"}, {"liquid_bulk", "Granel líquido"}, {"oversized", "Carga sobredimensionada"}}
}
func EquipmentChoices() []LoadChoice {
	return []LoadChoice{{"dry_van", "Dry Van"}, {"flatbed", "Flatbed"}, {"reefer", "Reefer"}, {"step_deck", "Step Deck"}, {"double_drop", "Double Drop"}, {"lowboy", "Lowboy"}, {"cargo_van", "Cargo Van"}, {"box_truck", "Box Truck"}, {"power_only", "Power Only"}}
}
