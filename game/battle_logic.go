package game

import (
	"math/rand"
	"time"

	icharacter "MyGame/Interface"
	"MyGame/Struct/Item"
)

type TurnHandler struct {
	rng *rand.Rand
}

func NewTurnHandler() *TurnHandler {
	return &TurnHandler{rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

func (th *TurnHandler) ExecuteAttack(attacker, defender icharacter.ICharacter, attackPart, blockPart string) bool {
	if attacker == nil || defender == nil || defender.GetHP() <= 0 {
		return false
	}
	if attackPart != blockPart {
		defender.TakeDamage(th.CalculateDamage(attacker, defender, attackPart))
	}
	return true
}

func (th *TurnHandler) SimpleStrike(attacker, defender icharacter.ICharacter) (damage int, blocked bool) {
	if attacker == nil || defender == nil || defender.GetHP() <= 0 {
		return 0, false
	}
	attackPart := attacker.Hit()
	blockPart := defender.Block()
	if attackPart == blockPart {
		return 0, true
	}
	damage = th.CalculateDamage(attacker, defender, attackPart)
	defender.TakeDamage(damage)
	return damage, false
}

func (th *TurnHandler) CalculateDamage(attacker, defender icharacter.ICharacter, bodyPart string) int {
	baseDamage := attacker.GetStrength() + int(attacker.GetAttack())
	defense := int(defender.GetDefense())

	m := bodyPartMultiplier(bodyPart)
	damage := float64(baseDamage)*m - float64(defense)*0.5

	if th.rng != nil {
		damage *= 0.9 + th.rng.Float64()*0.2
	}
	if damage < 1 {
		damage = 1
	}
	return int(damage)
}

func bodyPartMultiplier(bodyPart string) float64 {
	switch bodyPart {
	case "Голова":
		return 1.5
	case "Тело":
		return 1.0
	case "Правая нога", "Левая нога":
		return 0.8
	case "Правая рука", "Левая рука":
		return 0.9
	default:
		return 1.0
	}
}

type ItemEffect struct {
	Health   float32
	Mana     float32
	Attack   float32
	Defense  float32
	Duration int
}

type ItemEffectManager struct {
	effects map[int]ItemEffect
}

func NewItemEffectManager() *ItemEffectManager {
	return &ItemEffectManager{
		effects: map[int]ItemEffect{
			19: {Health: 50},
			22: {Mana: 30},
			23: {Attack: 10, Duration: 3},
		},
	}
}

func (iem *ItemEffectManager) GetUsableItems(items []*Item.Item) []*Item.Item {
	out := make([]*Item.Item, 0)
	for _, it := range items {
		if canUseInBattle(it) {
			out = append(out, it)
		}
	}
	return out
}

func canUseInBattle(item *Item.Item) bool {
	if item == nil || item.Template == nil || item.IsEquipped {
		return false
	}
	if item.Template.Slot != Item.SlotNone {
		return false
	}
	return item.Template.Type == Item.Potion || item.Template.Type == Item.Consumable
}

func (iem *ItemEffectManager) UseItem(player, enemy icharacter.ICharacter, item *Item.Item) bool {
	_ = enemy
	if player == nil || item == nil || item.Template == nil || !canUseInBattle(item) {
		return false
	}

	if iem != nil {
		if eff, ok := iem.effects[item.Template.ID]; ok {
			if eff.Health > 0 {
				player.Heal(eff.Health)
			}
			if eff.Mana > 0 {
				player.SetMana(player.GetMana() + eff.Mana)
			}
			if eff.Attack > 0 {
				player.SetAttack(player.GetAttack() + eff.Attack)
			}
			if eff.Defense > 0 {
				player.SetDefense(player.GetDefense() + eff.Defense)
			}
			return true
		}
	}

	effects, err := item.Use()
	if err != nil {
		return false
	}
	if v, ok := effects["health"]; ok && v > 0 {
		player.Heal(v)
	}
	if v, ok := effects["mana"]; ok && v > 0 {
		player.SetMana(player.GetMana() + v)
	}
	if v, ok := effects["attack"]; ok && v > 0 {
		player.SetAttack(player.GetAttack() + v)
	}
	if v, ok := effects["defense"]; ok && v > 0 {
		player.SetDefense(player.GetDefense() + v)
	}
	return true
}

func (iem *ItemEffectManager) GetItemEffectDescription(item *Item.Item) string {
	if item == nil || item.Template == nil {
		return "Особый эффект"
	}
	if iem != nil {
		if eff, ok := iem.effects[item.Template.ID]; ok {
			switch {
			case eff.Health > 0:
				return "Восстанавливает 50 HP"
			case eff.Mana > 0:
				return "Восстанавливает 30 MP"
			case eff.Attack > 0:
				return "Увеличивает атаку"
			case eff.Defense > 0:
				return "Увеличивает защиту"
			}
		}
	}
	if item.Health > 0 {
		return "Восстанавливает HP"
	}
	if item.Mana > 0 {
		return "Восстанавливает MP"
	}
	if item.Attack > 0 {
		return "Увеличивает атаку"
	}
	if item.Defense > 0 {
		return "Увеличивает защиту"
	}
	if item.Template.BaseHealth > 0 {
		return "Восстанавливает HP"
	}
	if item.Template.BaseMana > 0 {
		return "Восстанавливает MP"
	}
	return "Особый эффект"
}
