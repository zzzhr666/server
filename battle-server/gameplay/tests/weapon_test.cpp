#include "gameplay/weapon.hpp"

#include <gtest/gtest.h>

namespace battle {
namespace {

TEST(WeaponTest, MeleeDefinitionsUseTunedInitialDamageAndCooldown) {
    const auto sword = weapon_definition(WeaponKind::Sword);
    const auto dagger = weapon_definition(WeaponKind::Dagger);
    const auto axe = weapon_definition(WeaponKind::Axe);

    EXPECT_EQ(sword.attack.damage, 23);
    EXPECT_FLOAT_EQ(sword.attack.cooldown_seconds.count(), 0.24f);
    EXPECT_EQ(dagger.attack.damage, 13);
    EXPECT_FLOAT_EQ(dagger.attack.cooldown_seconds.count(), 0.13f);
    EXPECT_EQ(axe.attack.damage, 38);
    EXPECT_FLOAT_EQ(axe.attack.cooldown_seconds.count(), 0.48f);
}

TEST(WeaponTest, BowDefinitionUsesProjectileAttack) {
    const auto bow = weapon_definition(WeaponKind::Bow);

    EXPECT_EQ(bow.kind, WeaponKind::Bow);
    EXPECT_EQ(bow.attack.kind, ecs::AttackKind::Projectile);
    EXPECT_EQ(bow.attack.damage, 28);
    EXPECT_FLOAT_EQ(bow.attack.range, 15.0f);
    EXPECT_FLOAT_EQ(bow.attack.cooldown_seconds.count(), 0.32f);
    EXPECT_FLOAT_EQ(bow.attack.projectile_speed, 25.0f);
    EXPECT_FLOAT_EQ(bow.attack.projectile_hit_radius, 0.85f);
}

TEST(WeaponTest, BowStringCodecRoundTrips) {
    const auto parsed = weapon_kind_from_string("bow");

    ASSERT_TRUE(parsed.has_value());
    EXPECT_EQ(parsed.value(), WeaponKind::Bow);
    EXPECT_EQ(weapon_kind_to_string(WeaponKind::Bow), "bow");
}

}  // namespace
}  // namespace battle
