#include "gameplay/weapon.hpp"

#include <gtest/gtest.h>

namespace battle {
namespace {

TEST(WeaponTest, MeleeDefinitionsUseTunedInitialDamageAndCooldown) {
    const auto sword = weapon_definition(WeaponKind::Sword);
    const auto dagger = weapon_definition(WeaponKind::Dagger);
    const auto axe = weapon_definition(WeaponKind::Axe);

    EXPECT_EQ(sword.attack.damage, 23);
    EXPECT_FLOAT_EQ(sword.attack.cooldown_seconds.count(), 0.34f);
    EXPECT_FLOAT_EQ(sword.attack.windup_seconds.count(), 0.12f);
    EXPECT_FLOAT_EQ(sword.attack.active_seconds.count(), 0.05f);
    EXPECT_FLOAT_EQ(sword.attack.recovery_seconds.count(), 0.17f);
    EXPECT_FLOAT_EQ(sword.attack.movement_multiplier, 0.25f);
    EXPECT_EQ(dagger.attack.damage, 13);
    EXPECT_FLOAT_EQ(dagger.attack.cooldown_seconds.count(), 0.20f);
    EXPECT_FLOAT_EQ(dagger.attack.windup_seconds.count(), 0.06f);
    EXPECT_FLOAT_EQ(dagger.attack.active_seconds.count(), 0.03f);
    EXPECT_FLOAT_EQ(dagger.attack.recovery_seconds.count(), 0.11f);
    EXPECT_FLOAT_EQ(dagger.attack.movement_multiplier, 0.55f);
    EXPECT_EQ(axe.attack.damage, 38);
    EXPECT_FLOAT_EQ(axe.attack.cooldown_seconds.count(), 0.62f);
    EXPECT_FLOAT_EQ(axe.attack.windup_seconds.count(), 0.24f);
    EXPECT_FLOAT_EQ(axe.attack.active_seconds.count(), 0.08f);
    EXPECT_FLOAT_EQ(axe.attack.recovery_seconds.count(), 0.30f);
    EXPECT_FLOAT_EQ(axe.attack.movement_multiplier, 0.0f);
}

TEST(WeaponTest, BowDefinitionUsesProjectileAttack) {
    const auto bow = weapon_definition(WeaponKind::Bow);

    EXPECT_EQ(bow.kind, WeaponKind::Bow);
    EXPECT_EQ(bow.attack.kind, ecs::AttackKind::Projectile);
    EXPECT_EQ(bow.attack.damage, 32);
    EXPECT_FLOAT_EQ(bow.attack.range, 15.0f);
    EXPECT_FLOAT_EQ(bow.attack.cooldown_seconds.count(), 0.30f);
    EXPECT_FLOAT_EQ(bow.attack.windup_seconds.count(), 0.12f);
    EXPECT_FLOAT_EQ(bow.attack.active_seconds.count(), 0.02f);
    EXPECT_FLOAT_EQ(bow.attack.recovery_seconds.count(), 0.16f);
    EXPECT_FLOAT_EQ(bow.attack.movement_multiplier, 0.35f);
    EXPECT_FLOAT_EQ(bow.attack.projectile_speed, 25.0f);
    EXPECT_FLOAT_EQ(bow.attack.projectile_hit_radius, 0.85f);
}

TEST(WeaponTest, AttackTimelineDoesNotExceedCooldown) {
    for (const auto kind : {WeaponKind::Sword, WeaponKind::Dagger, WeaponKind::Axe, WeaponKind::Bow}) {
        const auto attack = weapon_definition(kind).attack;
        const float timeline_seconds = attack.windup_seconds.count() + attack.active_seconds.count() +
                                       attack.recovery_seconds.count();

        EXPECT_LE(timeline_seconds, attack.cooldown_seconds.count() + 0.000001f);
    }
}

TEST(WeaponTest, BowStringCodecRoundTrips) {
    const auto parsed = weapon_kind_from_string("bow");

    ASSERT_TRUE(parsed.has_value());
    EXPECT_EQ(parsed.value(), WeaponKind::Bow);
    EXPECT_EQ(weapon_kind_to_string(WeaponKind::Bow), "bow");
}

}  // namespace
}  // namespace battle
