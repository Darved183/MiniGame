package Inventory

import (
	"MyGame/Struct/Item"
	"fmt"
)

type Inventory struct {
	Items    []*Item.Item
	Capacity int
}

func NewInventory(capacity int) *Inventory {
	if capacity < 0 {
		capacity = 0
	}
	return &Inventory{
		Items:    make([]*Item.Item, 0),
		Capacity: capacity,
	}
}

func (inv *Inventory) AddItem(item *Item.Item) error {
	if item == nil {
		return fmt.Errorf("предмет не существует")
	}

	if len(inv.Items) >= inv.Capacity {
		return fmt.Errorf("инвентарь полон")
	}

	if item.IsEquipped {
		name := "?"
		if item.Template != nil {
			name = item.Template.Name
		}
		return fmt.Errorf("предмет '%s' уже экипирован", name)
	}

	inv.Items = append(inv.Items, item)
	return nil
}

func (inv *Inventory) RemoveItem(itemID int) (*Item.Item, error) {
	for i, item := range inv.Items {
		if item != nil && item.Template.ID == itemID {

			if item.IsEquipped {
				return nil, fmt.Errorf("нельзя удалить экипированный предмет")
			}

			removedItem := inv.Items[i]
			inv.Items = append(inv.Items[:i], inv.Items[i+1:]...)
			return removedItem, nil
		}
	}
	return nil, fmt.Errorf("предмет с ID %d не найден", itemID)
}

func (inv *Inventory) FindItemByID(itemID int) *Item.Item {
	for _, item := range inv.Items {
		if item != nil && item.Template != nil && item.Template.ID == itemID {
			return item
		}
	}
	return nil
}

func (inv *Inventory) FindItemsByType(itemType Item.ItemType) []*Item.Item {
	result := make([]*Item.Item, 0)
	for _, item := range inv.Items {
		if item != nil && item.Template.Type == itemType {
			result = append(result, item)
		}
	}
	return result
}

func (inv *Inventory) FindEquippableItems() []*Item.Item {
	result := make([]*Item.Item, 0)
	for _, item := range inv.Items {
		if item != nil && item.Template.Slot != Item.SlotNone && !item.IsEquipped {
			result = append(result, item)
		}
	}
	return result
}

func (inv *Inventory) GetItems() []*Item.Item {
	return inv.Items
}

func (inv *Inventory) IsFull() bool {
	return len(inv.Items) >= inv.Capacity
}

func (inv *Inventory) GetEmptySlots() int {
	empty := inv.Capacity - len(inv.Items)
	if empty < 0 {
		return 0
	}
	return empty
}

func (inv *Inventory) Display() {
	if len(inv.Items) == 0 {
		fmt.Println("Инвентарь пуст")
		return
	}

	fmt.Println("=== ИНВЕНТАРЬ ===")
	for i, item := range inv.Items {
		if item == nil {
			continue
		}

		equipped := ""
		if item.IsEquipped {
			equipped = " [Экипировано]"
		}
		fmt.Printf("%d. %s%s\n", i+1, item.Template.Name, equipped)
		fmt.Printf("   %s\n", item.Template.Description)

		stats := make([]string, 0)
		if item.Strength > 0 {
			stats = append(stats, fmt.Sprintf("Сила: +%d", item.Strength))
		}
		if item.Attack > 0 {
			stats = append(stats, fmt.Sprintf("Атака: +%.1f", item.Attack))
		}
		if item.Defense > 0 {
			stats = append(stats, fmt.Sprintf("Защита: +%.1f", item.Defense))
		}
		if item.Health > 0 {
			stats = append(stats, fmt.Sprintf("Здоровье: +%.1f", item.Health))
		}

		if len(stats) > 0 {
			fmt.Printf("   %s\n", joinStats(stats))
		}
	}
	fmt.Printf("Свободно мест: %d/%d\n", inv.GetEmptySlots(), inv.Capacity)
}

func (inv *Inventory) TransferItem(itemID int, targetInventory *Inventory) error {
	if targetInventory == nil {
		return fmt.Errorf("целевой инвентарь не существует")
	}

	item := inv.FindItemByID(itemID)
	if item == nil {
		return fmt.Errorf("предмет не найден")
	}

	if item.IsEquipped {
		return fmt.Errorf("нельзя перемещать экипированный предмет")
	}

	err := targetInventory.AddItem(item)
	if err != nil {
		return fmt.Errorf("не удалось переместить предмет: %w", err)
	}

	_, err = inv.RemoveItem(itemID)
	if err != nil {
		rollback, _ := targetInventory.RemoveItem(item.Template.ID)
		if rollback != nil {
			_ = inv.AddItem(rollback)
		}
		return fmt.Errorf("ошибка при перемещении: %w", err)
	}

	return nil
}

func joinStats(stats []string) string {
	result := ""
	for i, stat := range stats {
		if i > 0 {
			result += " "
		}
		result += stat
	}
	return result
}
