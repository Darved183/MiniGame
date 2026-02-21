package Item

import "fmt"

type ItemType int
type EquipmentSlot int
type Rarity int

const (
	Weapon ItemType = iota
	Potion
	Armor
	Accessory
	Consumable
)

const (
	SlotWeapon EquipmentSlot = iota
	SlotHead
	SlotBody
	SlotHands
	SlotFeet
	SlotAccessory
	SlotNone
	SlotBoots
)

const (
	Common Rarity = iota
	Uncommon
	Rare
	Epic
	Legendary
	Mythic
)

type ItemTemplate struct {
	ID               int
	Name             string
	Type             ItemType
	Slot             EquipmentSlot
	BaseStrength     int
	BaseAgility      int
	BaseIntelligence int
	BaseAttack       float32
	BaseDefense      float32
	BaseHealth       float32
	BaseMana         float32
	Description      string

	HealthRegen    float32
	ManaRegen      float32
	AttackSpeed    float32
	Evasion        float32
	CriticalChance float32
	MagicAmp       float32
	Lifesteal      float32
	Duration       int
}

type Item struct {
	Template *ItemTemplate
	Rarity   Rarity
	Level    int

	Strength     int
	Agility      int
	Intelligence int
	Attack       float32
	Defense      float32
	Health       float32
	Mana         float32

	Durability    int
	MaxDurability int
	Price         int
	IsEquipped    bool
}

var rarityMultipliers = map[Rarity]float64{
	Common:    1.0,
	Uncommon:  1.3,
	Rare:      1.7,
	Epic:      2.2,
	Legendary: 3.0,
	Mythic:    4.0,
}

const levelMultiplierPerLevel = 0.1

var rarityNames = map[Rarity]string{
	Common:    "Обычный",
	Uncommon:  "Необычный",
	Rare:      "Редкий",
	Epic:      "Эпический",
	Legendary: "Легендарный",
	Mythic:    "Мифический",
}

var typeNames = map[ItemType]string{
	Weapon:     "Оружие",
	Potion:     "Зелье",
	Armor:      "Броня",
	Accessory:  "Аксессуар",
	Consumable: "Расходный",
}

var itemTemplates = map[int]*ItemTemplate{
	1: {
		ID:           1,
		Name:         "Меч новичка",
		Type:         Weapon,
		Slot:         SlotWeapon,
		BaseAttack:   5.0,
		BaseStrength: 2,
		Description:  "Простой меч для начинающих воинов",
	},
	4: {
		ID:          4,
		Name:        "Кольцо Регенерации",
		Type:        Accessory,
		Slot:        SlotAccessory,
		BaseHealth:  10.0,
		HealthRegen: 1.0,
		Description: "Восстанавливает здоровье владельца",
	},
	19: {
		ID:          19,
		Name:        "Зелье здоровья",
		Type:        Potion,
		Slot:        SlotNone,
		BaseHealth:  50.0,
		Description: "Восстанавливает здоровье",
	},
	20: {
		ID:          20,
		Name:        "Свиток Возвращения",
		Type:        Consumable,
		Slot:        SlotNone,
		Description: "Возвращает в безопасное место",
	},
	21: {
		ID:          21,
		Name:        "Сторожевой Тотем",
		Type:        Accessory,
		Slot:        SlotAccessory,
		BaseDefense: 3.0,
		Description: "Защищает от неожиданных атак",
	},
	22: {
		ID:          22,
		Name:        "Зелье маны",
		Type:        Potion,
		Slot:        SlotNone,
		BaseMana:    30.0,
		Description: "Восстанавливает ману",
	},
	23: {
		ID:          23,
		Name:        "Зелье силы",
		Type:        Potion,
		Slot:        SlotNone,
		BaseAttack:  10.0,
		Duration:    3,
		Description: "Временно увеличивает атаку",
	},
	24: {
		ID:          24,
		Name:        "Быстрый клинок",
		Type:        Weapon,
		Slot:        SlotWeapon,
		BaseAttack:  3.0,
		BaseAgility: 4,
		Description: "Лёгкий клинок, повышает ловкость",
	},
	25: {
		ID:           25,
		Name:         "Тяжёлый палаш",
		Type:         Weapon,
		Slot:         SlotWeapon,
		BaseAttack:   7.0,
		BaseStrength: 1,
		Description:  "Мощный удар, но медленный",
	},
	26: {
		ID:          26,
		Name:        "Защитный клинок",
		Type:        Weapon,
		Slot:        SlotWeapon,
		BaseAttack:  4.0,
		BaseDefense: 2.0,
		Description: "Баланс атаки и защиты",
	},
}

func CreateItem(templateID int, rarity Rarity, level int) (*Item, error) {
	template, exists := itemTemplates[templateID]
	if !exists {
		return nil, fmt.Errorf("шаблон предмета с ID %d не найден", templateID)
	}

	if level < 1 {
		return nil, fmt.Errorf("уровень предмета должен быть положительным")
	}

	if rarity < Common || rarity > Mythic {
		return nil, fmt.Errorf("некорректная редкость предмета")
	}

	rarityMultiplier := rarityMultipliers[rarity]
	levelMultiplier := 1.0 + (float64(level-1) * levelMultiplierPerLevel)

	item := &Item{
		Template:      template,
		Rarity:        rarity,
		Level:         level,
		Durability:    100,
		MaxDurability: 100,
		IsEquipped:    false,
	}

	totalMultiplier := rarityMultiplier * levelMultiplier

	item.Strength = int(float64(template.BaseStrength) * totalMultiplier)
	item.Agility = int(float64(template.BaseAgility) * totalMultiplier)
	item.Intelligence = int(float64(template.BaseIntelligence) * totalMultiplier)
	item.Attack = float32(float64(template.BaseAttack) * totalMultiplier)
	item.Defense = float32(float64(template.BaseDefense) * totalMultiplier)
	item.Health = float32(float64(template.BaseHealth) * totalMultiplier)
	item.Mana = float32(float64(template.BaseMana) * totalMultiplier)

	basePrice := 10 + (level * 5)
	item.Price = int(float64(basePrice) * totalMultiplier)

	return item, nil
}

func (i *Item) GetFullName() string {
	if i.Template == nil {
		return "Неизвестный предмет"
	}
	rarityName := rarityNames[i.Rarity]
	return fmt.Sprintf("%s %s", rarityName, i.Template.Name)
}

func (i *Item) GetTypeName() string {
	if i.Template == nil {
		return "Неизвестно"
	}
	return typeNames[i.Template.Type]
}

func (i *Item) GetDetailedDescription() string {
	if i.Template == nil {
		return "Неизвестный предмет"
	}

	desc := fmt.Sprintf("%s\n", i.GetFullName())
	desc += fmt.Sprintf("Тип: %s\n", i.GetTypeName())

	if i.Template.Slot != SlotNone {
		desc += fmt.Sprintf("Слот: %s\n", i.GetSlotName())
	}

	if i.Strength != 0 {
		desc += fmt.Sprintf("Сила: +%d\n", i.Strength)
	}
	if i.Agility != 0 {
		desc += fmt.Sprintf("Ловкость: +%d\n", i.Agility)
	}
	if i.Intelligence != 0 {
		desc += fmt.Sprintf("Интеллект: +%d\n", i.Intelligence)
	}
	if i.Attack != 0 {
		desc += fmt.Sprintf("Атака: +%.1f\n", i.Attack)
	}
	if i.Defense != 0 {
		desc += fmt.Sprintf("Защита: +%.1f\n", i.Defense)
	}
	if i.Health != 0 {
		desc += fmt.Sprintf("Здоровье: +%.1f\n", i.Health)
	}
	if i.Mana != 0 {
		desc += fmt.Sprintf("Мана: +%.1f\n", i.Mana)
	}
	if i.Template.HealthRegen != 0 {
		desc += fmt.Sprintf("Регенерация здоровья: +%.1f/сек\n", i.Template.HealthRegen)
	}
	if i.Template.ManaRegen != 0 {
		desc += fmt.Sprintf("Регенерация маны: +%.1f/сек\n", i.Template.ManaRegen)
	}
	if i.Template.CriticalChance != 0 {
		desc += fmt.Sprintf("Шанс критического удара: +%.1f%%\n", i.Template.CriticalChance*100)
	}
	if i.Template.Lifesteal != 0 {
		desc += fmt.Sprintf("Вампиризм: +%.1f%%\n", i.Template.Lifesteal*100)
	}

	desc += fmt.Sprintf("Уровень: %d\n", i.Level)
	desc += fmt.Sprintf("Прочность: %d/%d\n", i.Durability, i.MaxDurability)
	desc += fmt.Sprintf("Цена: %d золотых\n", i.Price)

	if i.IsEquipped {
		desc += "Состояние: Экипировано\n"
	}

	desc += fmt.Sprintf("Описание: %s", i.Template.Description)

	return desc
}

func (i *Item) GetSlotName() string {
	if i.Template == nil {
		return "Неизвестно"
	}

	slotNames := map[EquipmentSlot]string{
		SlotWeapon:    "Оружие",
		SlotHead:      "Голова",
		SlotBody:      "Тело",
		SlotHands:     "Руки",
		SlotFeet:      "Ноги",
		SlotBoots:     "Обувь",
		SlotAccessory: "Аксессуар",
		SlotNone:      "Не экипируется",
	}
	return slotNames[i.Template.Slot]
}

func (i *Item) CanUse() bool {
	if i.Template == nil {
		return false
	}

	switch i.Template.Type {
	case Potion, Consumable:
		return true
	default:
		return false
	}
}

func (i *Item) Use() (map[string]float32, error) {
	if !i.CanUse() {
		return nil, fmt.Errorf("этот предмет нельзя использовать")
	}

	if i.Durability <= 0 {
		return nil, fmt.Errorf("предмет сломан и не может быть использован")
	}

	effects := make(map[string]float32)

	if i.Health != 0 {
		effects["health"] = i.Health
	}
	if i.Mana != 0 {
		effects["mana"] = i.Mana
	}
	if i.Attack != 0 {
		effects["attack"] = i.Attack
	}
	if i.Defense != 0 {
		effects["defense"] = i.Defense
	}

	if i.Template.HealthRegen != 0 {
		effects["health_regen"] = i.Template.HealthRegen
	}
	if i.Template.ManaRegen != 0 {
		effects["mana_regen"] = i.Template.ManaRegen
	}

	switch i.Template.Type {
	case Consumable:
		i.Durability = 0
	case Potion:
		i.Durability--
	}

	return effects, nil
}

func CreateSword() *Item {
	item, err := CreateItem(1, Common, 1)
	if err != nil {
		return nil
	}
	return item
}

func CreatePvPSword(templateID int) *Item {
	item, err := CreateItem(templateID, Common, 1)
	if err != nil {
		return nil
	}
	return item
}

func CreateHealthPotion() *Item {
	item, err := CreateItem(19, Common, 1)
	if err != nil {
		return nil
	}
	return item
}
