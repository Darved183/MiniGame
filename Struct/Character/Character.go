package Character

import (
	"fmt"
	"math/rand"
	"time"

	icharacter "MyGame/Interface"
	"MyGame/Struct/Equipment"
	"MyGame/Struct/Inventory"
	"MyGame/Struct/Item"
)

const (
	MinHP           = 0
	MinStrength     = 1
	MinAgility      = 1
	MinIntelligence = 1
)

type Character struct {
	Name         string
	CurrentHP    int
	MaxHP        int
	Strength     int
	Agility      int
	Intelligence int
	Description  string

	BaseHP           int
	BaseStrength     int
	BaseAgility      int
	BaseIntelligence int

	Inventory *Inventory.Inventory
	Equipment *Equipment.Equipment

	AttackValue  float32
	DefenseValue float32
	AttackSpeed  float32
	CritChance   float32
	CritDamage   float32
	HealthRegen  float32
	Mana         float32
	MaxMana      float32
	ManaRegen    float32
}

var (
	bodyParts = []string{"Голова", "Тело", "Правая нога", "Левая нога", "Правая рука", "Левая рука"}
	rng       = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func New(name string, hp, strength, agility, intelligence int) (*Character, error) {
	if err := validateBaseStats(hp, strength, agility, intelligence); err != nil {
		return nil, err
	}

	char := &Character{
		Name:             name,
		CurrentHP:        hp,
		MaxHP:            hp,
		Strength:         strength,
		Agility:          agility,
		Intelligence:     intelligence,
		BaseHP:           hp,
		BaseStrength:     strength,
		BaseAgility:      agility,
		BaseIntelligence: intelligence,
		Inventory:        Inventory.NewInventory(20),
		Equipment:        Equipment.NewEquipment(),
		AttackValue:      0,
		DefenseValue:     0,
		CritChance:       0.1,
		CritDamage:       1.5,
		HealthRegen:      0.5,
		MaxMana:          100,
		Mana:             100,
		ManaRegen:        1.0,
	}

	char.AddStarterItems()
	char.CalculateStats()

	return char, nil
}
func validateBaseStats(hp, strength, agility, intelligence int) error {
	if hp < MinHP {
		return fmt.Errorf("HP не может быть меньше, чем %d", MinHP)
	}
	if strength < MinStrength {
		return fmt.Errorf("сила не может быть меньше, чем %d", MinStrength)
	}
	if agility < MinAgility {
		return fmt.Errorf("ловкость не может быть меньше, чем %d", MinAgility)
	}
	if intelligence < MinIntelligence {
		return fmt.Errorf("интеллект не может быть меньше, чем %d", MinIntelligence)
	}
	return nil
}

func NewWarrior(name string) (*Character, error) {
	return New(name, 100, 15, 8, 5)
}

func NewMage(name string) (*Character, error) {
	return New(name, 70, 5, 8, 15)
}

func NewRogue(name string) (*Character, error) {
	return New(name, 80, 8, 15, 5)
}

func (c *Character) AddStarterItems() {
	if sword := Item.CreateSword(); sword != nil {
		c.Inventory.AddItem(sword)
	}
	if potion := Item.CreateHealthPotion(); potion != nil {
		c.Inventory.AddItem(potion)
	}
}

func (c *Character) Hit() string {
	return bodyParts[rng.Intn(len(bodyParts))]
}

func (c *Character) Block() string {
	return bodyParts[rng.Intn(len(bodyParts))]
}

func (c *Character) Attack(target icharacter.ICharacter, attackPart, blockPart string) {
	if !c.IsAlive() {
		fmt.Printf("%s не может атаковать, так как мертв\n", c.Name)
		return
	}

	if !target.IsAlive() {
		fmt.Printf("%s не может атаковать мертвого %s\n", c.Name, target.GetName())
		return
	}

	if attackPart != blockPart {
		baseDamage := c.Strength + int(c.GetAttack())

		isCritical := rng.Float32() < c.CritChance
		damage := baseDamage
		if isCritical {
			damage = int(float32(baseDamage) * c.CritDamage)
			fmt.Printf("КРИТИЧЕСКИЙ УДАР! ")
		}

		target.TakeDamage(damage)
		fmt.Printf("%s атакует %s в %s и наносит %d урона. %s HP = %d/%d\n",
			c.Name, target.GetName(), attackPart, damage, target.GetName(), target.GetHP(), target.GetMaxHP())
	} else {
		fmt.Printf("%s блокирует атаку %s в %s и урон не наносится\n",
			target.GetName(), c.Name, attackPart)
	}
}

func (c *Character) GetHP() int {
	return c.CurrentHP
}

func (c *Character) GetMaxHP() int {
	return c.MaxHP
}

func (c *Character) GetName() string {
	return c.Name
}

func (c *Character) GetStrength() int {
	return c.Strength
}

func (c *Character) GetAgility() int {
	return c.Agility
}

func (c *Character) GetIntelligence() int {
	return c.Intelligence
}

func (c *Character) TakeDamage(damage int) {
	if damage < 0 {
		damage = 0
	}

	if damage == 0 {
		return
	}

	dodgeChance := float32(c.Agility) * 0.01
	if rng.Float32() < dodgeChance {
		fmt.Printf("%s увернулся от атаки!\n", c.Name)
		return
	}

	actualDamage := damage - int(c.DefenseValue)

	if actualDamage < 1 {
		actualDamage = 1
	}

	c.CurrentHP = max(MinHP, c.CurrentHP-actualDamage)

	if c.CurrentHP == 0 {
		fmt.Printf("%s погиб!\n", c.Name)
	}
}

func (c *Character) Heal(amount float32) {
	if amount <= 0 {
		return
	}

	healAmount := int(amount)
	oldHP := c.CurrentHP
	c.CurrentHP = min(c.CurrentHP+healAmount, c.MaxHP)
	actualHeal := c.CurrentHP - oldHP

	if actualHeal > 0 {
		fmt.Printf("%s восстанавливает %d HP (теперь %d/%d)\n",
			c.Name, actualHeal, c.CurrentHP, c.MaxHP)
	}
}

func (c *Character) SetHP(hp int) {
	if hp < MinHP {
		c.CurrentHP = MinHP
	} else if hp > c.MaxHP {
		c.CurrentHP = c.MaxHP
	} else {
		c.CurrentHP = hp
	}
}

func (c *Character) SetAttack(attack float32) {
	if attack < 0 {
		attack = 0
	}
	c.AttackValue = attack
}

func (c *Character) GetAttack() float32 {
	return c.AttackValue
}

func (c *Character) GetDefense() float32 {
	return c.DefenseValue
}

func (c *Character) SetDefense(defense float32) {
	if defense < 0 {
		defense = 0
	}
	c.DefenseValue = defense
}

func (c *Character) GetMana() float32 {
	return c.Mana
}

func (c *Character) GetMaxMana() float32 {
	return c.MaxMana
}

func (c *Character) SetMana(mana float32) {
	if mana < 0 {
		mana = 0
	}
	if mana > c.MaxMana {
		mana = c.MaxMana
	}
	c.Mana = mana
}

func (c *Character) GetInventory() *Inventory.Inventory {
	return c.Inventory
}

func (c *Character) GetEquipment() *Equipment.Equipment {
	return c.Equipment
}

func (c *Character) EquipItem(itemID int) error {
	item := c.Inventory.FindItemByID(itemID)
	if item == nil {
		return fmt.Errorf("предмет с ID %d не найден в инвентаре", itemID)
	}

	if !c.Equipment.CanEquipItem(item) {
		return fmt.Errorf("предмет '%s' нельзя экипировать", item.Template.Name)
	}

	err := c.Equipment.Equip(item)
	if err != nil {
		return err
	}

	_, err = c.Inventory.RemoveItem(itemID)
	if err != nil {
		c.Equipment.Unequip(item.Template.Slot)
		return fmt.Errorf("не удалось удалить предмет из инвентаря: %w", err)
	}

	c.CalculateStats()
	fmt.Printf("%s экипировал %s\n", c.Name, item.Template.Name)
	return nil
}

func (c *Character) UnequipItem(slot Item.EquipmentSlot) error {
	item, err := c.Equipment.Unequip(slot)
	if err != nil {
		return err
	}

	err = c.Inventory.AddItem(item)
	if err != nil {
		c.Equipment.Equip(item)
		return fmt.Errorf("не удалось добавить предмет в инвентарь: %w", err)
	}

	c.CalculateStats()
	fmt.Printf("%s снял %s\n", c.Name, item.Template.Name)
	return nil
}

func (c *Character) CalculateStats() {
	c.MaxHP = c.BaseHP
	c.Strength = c.BaseStrength
	c.Agility = c.BaseAgility
	c.Intelligence = c.BaseIntelligence
	c.AttackValue = 0
	c.DefenseValue = 0
	c.CritChance = 0.05
	c.CritDamage = 1.5

	bonuses := c.Equipment.GetTotalBonuses()

	c.Strength += bonuses.Strength
	c.Agility += bonuses.Agility
	c.Intelligence += bonuses.Intelligence
	c.AttackValue += bonuses.Attack
	c.DefenseValue += bonuses.Defense
	c.MaxHP += int(bonuses.Health)

	c.CritChance += float32(c.Agility) * 0.001

	if c.CurrentHP > c.MaxHP {
		c.CurrentHP = c.MaxHP
	}
}

func (c *Character) ShowCharacterInfo() {
	fmt.Printf("\n=== ИНФОРМАЦИЯ О ПЕРСОНАЖЕ ===\n")
	fmt.Printf("Имя: %s\n", c.Name)
	fmt.Printf("Состояние: %s\n", c.GetStatus())
	fmt.Printf("Здоровье: %d/%d\n", c.CurrentHP, c.MaxHP)
	fmt.Printf("Сила: %d\n", c.Strength)
	fmt.Printf("Ловкость: %d\n", c.Agility)
	fmt.Printf("Интеллект: %d\n", c.Intelligence)
	fmt.Printf("Атака: %.1f\n", c.AttackValue)
	fmt.Printf("Защита: %.1f\n", c.DefenseValue)
	fmt.Printf("Шанс крита: %.1f%%\n", c.CritChance*100)
	fmt.Printf("Урон крита: %.1f%%\n", c.CritDamage*100)
	fmt.Printf("Мана: %.1f/%.1f\n", c.Mana, c.MaxMana)

	c.Equipment.Display()
	fmt.Println()
}

func (c *Character) GetStatus() string {
	if !c.IsAlive() {
		return "Мертв"
	}

	hpPercent := float32(c.CurrentHP) / float32(c.MaxHP) * 100
	switch {
	case hpPercent >= 80:
		return "Здоров"
	case hpPercent >= 50:
		return "Ранен"
	case hpPercent >= 25:
		return "Тяжело ранен"
	default:
		return "При смерти"
	}
}

func (c *Character) IsAlive() bool {
	return c.CurrentHP > 0
}

func (c *Character) UseItem(itemID int) error {
	item := c.Inventory.FindItemByID(itemID)
	if item == nil {
		return fmt.Errorf("предмет с ID %d не найден", itemID)
	}

	if !item.CanUse() {
		return fmt.Errorf("этот предмет нельзя использовать")
	}

	effects, err := item.Use()
	if err != nil {
		return err
	}

	for effect, value := range effects {
		switch effect {
		case "health":
			c.Heal(value)
		case "mana":
			c.Mana = minFloat32(c.Mana+value, c.MaxMana)
			fmt.Printf("%s восстанавливает %.1f маны (теперь %.1f/%.1f)\n", c.Name, value, c.Mana, c.MaxMana)
		case "attack", "defense":

			fmt.Printf("%s получает временный бафф %s: +%.1f\n", c.Name, effect, value)
		}
	}

	if item.Durability <= 0 {
		if _, err := c.Inventory.RemoveItem(itemID); err != nil {
			fmt.Printf("не удалось удалить использованный предмет из инвентаря: %v\n", err)
		}
	}

	return nil
}

func (c *Character) GetBaseHP() int {
	return c.BaseHP
}

func (c *Character) GetBaseStrength() int {
	return c.BaseStrength
}

func (c *Character) GetBaseAgility() int {
	return c.BaseAgility
}

func (c *Character) GetBaseIntelligence() int {
	return c.BaseIntelligence
}

func minFloat32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
