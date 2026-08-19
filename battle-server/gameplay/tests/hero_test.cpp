#include "gameplay/hero.hpp"

#include <string_view>
#include <utility>

#include <gtest/gtest.h>

namespace battle {
namespace {

TEST(HeroTest, MeleeDefinitionsUseTunedInitialDamageAndCooldown) {
    const auto fire = hero_definition(HeroKind::Fire);
    const auto ice = hero_definition(HeroKind::Ice);
    const auto rock = hero_definition(HeroKind::Rock);

    EXPECT_EQ(fire.attack.damage, 23);
    EXPECT_FLOAT_EQ(fire.attack.cooldown_seconds.count(), 0.34f);
    EXPECT_FLOAT_EQ(fire.attack.windup_seconds.count(), 0.12f);
    EXPECT_FLOAT_EQ(fire.attack.active_seconds.count(), 0.05f);
    EXPECT_FLOAT_EQ(fire.attack.recovery_seconds.count(), 0.17f);
    EXPECT_FLOAT_EQ(fire.attack.movement_multiplier, 0.25f);
    EXPECT_EQ(ice.attack.damage, 13);
    EXPECT_FLOAT_EQ(ice.attack.cooldown_seconds.count(), 0.20f);
    EXPECT_FLOAT_EQ(ice.attack.windup_seconds.count(), 0.06f);
    EXPECT_FLOAT_EQ(ice.attack.active_seconds.count(), 0.03f);
    EXPECT_FLOAT_EQ(ice.attack.recovery_seconds.count(), 0.11f);
    EXPECT_FLOAT_EQ(ice.attack.movement_multiplier, 0.55f);
    EXPECT_EQ(rock.attack.damage, 38);
    EXPECT_FLOAT_EQ(rock.attack.cooldown_seconds.count(), 0.62f);
    EXPECT_FLOAT_EQ(rock.attack.windup_seconds.count(), 0.24f);
    EXPECT_FLOAT_EQ(rock.attack.active_seconds.count(), 0.08f);
    EXPECT_FLOAT_EQ(rock.attack.recovery_seconds.count(), 0.30f);
    EXPECT_FLOAT_EQ(rock.attack.movement_multiplier, 0.0f);
}

TEST(HeroTest, NatureDefinitionUsesProjectileAttack) {
    const auto nature = hero_definition(HeroKind::Nature);

    EXPECT_EQ(nature.kind, HeroKind::Nature);
    EXPECT_EQ(nature.attack.kind, ecs::AttackKind::Projectile);
    EXPECT_EQ(nature.attack.damage, 32);
    EXPECT_FLOAT_EQ(nature.attack.range, 15.0f);
    EXPECT_FLOAT_EQ(nature.attack.cooldown_seconds.count(), 0.30f);
    EXPECT_FLOAT_EQ(nature.attack.windup_seconds.count(), 0.12f);
    EXPECT_FLOAT_EQ(nature.attack.active_seconds.count(), 0.02f);
    EXPECT_FLOAT_EQ(nature.attack.recovery_seconds.count(), 0.16f);
    EXPECT_FLOAT_EQ(nature.attack.movement_multiplier, 0.35f);
    EXPECT_FLOAT_EQ(nature.attack.projectile_speed, 25.0f);
    EXPECT_FLOAT_EQ(nature.attack.projectile_hit_radius, 0.85f);
}

TEST(HeroTest, AttackTimelineDoesNotExceedCooldown) {
    for (const auto kind : {HeroKind::Fire, HeroKind::Ice, HeroKind::Rock, HeroKind::Nature}) {
        const auto attack = hero_definition(kind).attack;
        const float timeline_seconds = attack.windup_seconds.count() + attack.active_seconds.count() +
                                       attack.recovery_seconds.count();

        EXPECT_LE(timeline_seconds, attack.cooldown_seconds.count() + 0.000001f);
    }
}

TEST(HeroTest, StringCodecRoundTripsEveryHero) {
    for (const auto& [text, kind] : {
             std::pair<std::string_view, HeroKind>{"fire", HeroKind::Fire},
             std::pair<std::string_view, HeroKind>{"ice", HeroKind::Ice},
             std::pair<std::string_view, HeroKind>{"rock", HeroKind::Rock},
             std::pair<std::string_view, HeroKind>{"nature", HeroKind::Nature},
         }) {
        const auto parsed = hero_kind_from_string(text);

        ASSERT_TRUE(parsed.has_value());
        EXPECT_EQ(parsed.value(), kind);
        EXPECT_EQ(hero_kind_to_string(kind), text);
    }
}

TEST(HeroTest, StringCodecRejectsLegacyWeaponNames) {
    for (const auto value : {"sword", "dagger", "axe", "bow"}) {
        EXPECT_FALSE(hero_kind_from_string(value).has_value());
    }
}

}  // namespace
}  // namespace battle
