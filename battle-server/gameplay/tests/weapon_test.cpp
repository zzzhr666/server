#include "gameplay/weapon.hpp"

#include <gtest/gtest.h>

namespace battle {
namespace {

TEST(WeaponTest, BowDefinitionUsesProjectileAttack) {
    const auto bow = weapon_definition(WeaponKind::Bow);

    EXPECT_EQ(bow.kind, WeaponKind::Bow);
    EXPECT_EQ(bow.attack.kind, ecs::AttackKind::Projectile);
    EXPECT_EQ(bow.attack.damage, 18);
    EXPECT_FLOAT_EQ(bow.attack.range, 15.0f);
    EXPECT_FLOAT_EQ(bow.attack.cooldown_seconds.count(), 0.45f);
    EXPECT_FLOAT_EQ(bow.attack.projectile_speed, 25.0f);
}

TEST(WeaponTest, BowStringCodecRoundTrips) {
    const auto parsed = weapon_kind_from_string("bow");

    ASSERT_TRUE(parsed.has_value());
    EXPECT_EQ(parsed.value(), WeaponKind::Bow);
    EXPECT_EQ(weapon_kind_to_string(WeaponKind::Bow), "bow");
}

}  // namespace
}  // namespace battle
