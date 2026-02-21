package Equipment

import (
	"MyGame/Struct/Item"
	"fmt"
)

type Equipment struct {
	Slots map[Item.EquipmentSlot]*Item.Item
}

func NewEquipment() *Equipment {
	return &Equipment{
		Slots: make(map[Item.EquipmentSlot]*Item.Item),
	}
}

func (e *Equipment) Equip(item *Item.Item) error {
	if item == nil || item.Template == nil {
		return fmt.Errorf("предмет не существует")
	}

	if item.Template.Slot == Item.SlotNone {
		return fmt.Errorf("этот предмет нельзя экипировать")
	}

	if !e.isValidItemForSlot(item, item.Template.Slot) {
		return fmt.Errorf("предмет типа '%s' нельзя экипировать в слот '%s'",
			item.GetTypeName(), e.GetSlotName(item.Template.Slot))
	}

	if currentItem, exists := e.Slots[item.Template.Slot]; exists && currentItem != nil {
		return fmt.Errorf("слот %s уже занят предметом: %s",
			e.GetSlotName(item.Template.Slot), currentItem.Template.Name)
	}

	item.IsEquipped = true
	e.Slots[item.Template.Slot] = item
	return nil
}

func (e *Equipment) Unequip(slot Item.EquipmentSlot) (*Item.Item, error) {
	item, exists := e.Slots[slot]
	if !exists || item == nil {
		return nil, fmt.Errorf("слот %s пуст", e.GetSlotName(slot))
	}

	item.IsEquipped = false
	delete(e.Slots, slot)
	return item, nil
}

func (e *Equipment) GetItem(slot Item.EquipmentSlot) *Item.Item {
	return e.Slots[slot]
}

func (e *Equipment) Display() {
	fmt.Println("=== ЭКИПИРОВКА ===")
	slots := e.GetAllEquipmentSlots()

	for _, slot := range slots {
		item := e.Slots[slot]
		itemName := "Пусто"
		if item != nil && item.Template != nil {
			itemName = fmt.Sprintf("%s (%s)", item.Template.Name, item.GetTypeName())
		}
		fmt.Printf("%s: %s\n", e.GetSlotName(slot), itemName)
	}
}

func (e *Equipment) GetSlotName(slot Item.EquipmentSlot) string {
	names := map[Item.EquipmentSlot]string{
		Item.SlotWeapon:    "Оружие",
		Item.SlotHead:      "Голова",
		Item.SlotBody:      "Тело",
		Item.SlotHands:     "Руки",
		Item.SlotFeet:      "Ноги",
		Item.SlotBoots:     "Обувь",
		Item.SlotAccessory: "Аксессуар",
		Item.SlotNone:      "Не экипируется",
	}
	return names[slot]
}

func (e *Equipment) GetTotalBonuses() *Item.Item {
	total := &Item.Item{}

	for _, item := range e.Slots {
		if item != nil {
			total.Strength += item.Strength
			total.Agility += item.Agility
			total.Intelligence += item.Intelligence
			total.Attack += item.Attack
			total.Defense += item.Defense
			total.Health += item.Health
			total.Mana += item.Mana
		}
	}

	return total
}

func (e *Equipment) GetAllEquipmentSlots() []Item.EquipmentSlot {
	return []Item.EquipmentSlot{
		Item.SlotWeapon,
		Item.SlotHead,
		Item.SlotBody,
		Item.SlotHands,
		Item.SlotFeet,
		Item.SlotBoots,
		Item.SlotAccessory,
	}
}

func (e *Equipment) GetOccupiedSlots() []Item.EquipmentSlot {
	occupied := make([]Item.EquipmentSlot, 0, len(e.Slots))
	for slot, item := range e.Slots {
		if item != nil {
			occupied = append(occupied, slot)
		}
	}
	return occupied
}

func (e *Equipment) GetEmptySlots() []Item.EquipmentSlot {
	allSlots := e.GetAllEquipmentSlots()
	empty := make([]Item.EquipmentSlot, 0, len(allSlots))

	for _, slot := range allSlots {
		if _, exists := e.Slots[slot]; !exists {
			empty = append(empty, slot)
		}
	}
	return empty
}

func (e *Equipment) isValidItemForSlot(item *Item.Item, slot Item.EquipmentSlot) bool {
	if item == nil || item.Template == nil {
		return false
	}

	if item.Template.Slot == Item.SlotNone {
		return false
	}

	switch slot {
	case Item.SlotWeapon:
		return item.Template.Type == Item.Weapon
	case Item.SlotHead, Item.SlotBody, Item.SlotHands, Item.SlotFeet, Item.SlotBoots:
		return item.Template.Type == Item.Armor
	case Item.SlotAccessory:
		return item.Template.Type == Item.Accessory
	default:
		return false
	}
}

func (e *Equipment) CanEquipItem(item *Item.Item) bool {
	if item == nil || item.Template == nil {
		return false
	}

	if item.Template.Slot == Item.SlotNone {
		return false
	}

	if currentItem, exists := e.Slots[item.Template.Slot]; exists && currentItem != nil {
		return false
	}

	return e.isValidItemForSlot(item, item.Template.Slot)
}
