package icharacter

import (
	"MyGame/Struct/Equipment"
	"MyGame/Struct/Inventory"
	"MyGame/Struct/Item"
)

type ICharacter interface {

	GetHP() int
	GetMaxHP() int
	GetName() string
	GetStrength() int
	GetAgility() int
	GetIntelligence() int
	GetAttack() float32
	GetDefense() float32


	TakeDamage(damage int)
	Heal(amount float32)
	Hit() string
	Block() string
	Attack(target ICharacter, attackPart, blockPart string)

	SetAttack(attack float32)
	SetDefense(defense float32)

	GetInventory() *Inventory.Inventory
	GetEquipment() *Equipment.Equipment
	EquipItem(itemID int) error
	UnequipItem(slot Item.EquipmentSlot) error

	CalculateStats()
	ShowCharacterInfo()

	GetBaseHP() int
	GetBaseStrength() int
	GetBaseAgility() int
	GetBaseIntelligence() int

	IsAlive() bool

	GetMana() float32
	GetMaxMana() float32
	SetMana(mana float32)
}
