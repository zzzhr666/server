#pragma once

#include <cstddef>

#include "ecs/time.hpp"

namespace battle::gameplay_config {
    namespace combat {
        inline constexpr float DefaultProjectileHitRadius = 0.5f;
        inline constexpr ecs::DeltaTime DefaultAttackActiveSeconds{0.05f};
        inline constexpr float DefaultCharacterCollisionRadius = 0.5f;
        inline constexpr float MonsterCollisionRadius = 0.4f;
    }

    namespace player {
        inline constexpr int MaxHealth = 500;
        inline constexpr float MoveSpeed = 11.0f;
        inline constexpr int AttackDamage = 23;
        inline constexpr float AttackRange = 3.0f;
        inline constexpr ecs::DeltaTime AttackCooldown{0.24f};
        inline constexpr float DashSpeedMultiplier = 25.0f;
        inline constexpr ecs::DeltaTime DashCooldown{1.0f};
    }

    namespace hero {
        namespace fire {
            inline constexpr int AttackDamage = 23;
            inline constexpr float AttackRange = 3.0f;
            inline constexpr ecs::DeltaTime AttackCooldown{0.34f};
            inline constexpr ecs::DeltaTime AttackWindup{0.12f};
            inline constexpr ecs::DeltaTime AttackActive{0.05f};
            inline constexpr ecs::DeltaTime AttackRecovery{0.17f};
            inline constexpr float AttackMovementMultiplier = 0.25f;
        }

        namespace ice {
            inline constexpr int AttackDamage = 13;
            inline constexpr float AttackRange = 2.4f;
            inline constexpr ecs::DeltaTime AttackCooldown{0.20f};
            inline constexpr ecs::DeltaTime AttackWindup{0.06f};
            inline constexpr ecs::DeltaTime AttackActive{0.03f};
            inline constexpr ecs::DeltaTime AttackRecovery{0.11f};
            inline constexpr float AttackMovementMultiplier = 0.55f;
        }

        namespace rock {
            inline constexpr int AttackDamage = 38;
            inline constexpr float AttackRange = 3.5f;
            inline constexpr ecs::DeltaTime AttackCooldown{0.62f};
            inline constexpr ecs::DeltaTime AttackWindup{0.24f};
            inline constexpr ecs::DeltaTime AttackActive{0.08f};
            inline constexpr ecs::DeltaTime AttackRecovery{0.30f};
            inline constexpr float AttackMovementMultiplier = 0.0f;
        }

        namespace nature {
            inline constexpr int AttackDamage = 32;
            inline constexpr float AttackRange = 15.0f;
            inline constexpr ecs::DeltaTime AttackCooldown{0.30f};
            inline constexpr ecs::DeltaTime AttackWindup{0.12f};
            inline constexpr ecs::DeltaTime AttackActive{0.02f};
            inline constexpr ecs::DeltaTime AttackRecovery{0.16f};
            inline constexpr float AttackMovementMultiplier = 0.35f;
            inline constexpr float ProjectileSpeed = 25.0f;
            inline constexpr float ProjectileHitRadius = 0.85f;
        }
    }

    namespace growth {
        inline constexpr float AttackIncreasePerLevel = 0.08f;
        inline constexpr float AttackSpeedIncreasePerLevel = 0.06f;
        inline constexpr float HealthIncreasePerLevel = 0.1f;
        inline constexpr float MoveSpeedIncreasePerLevel = 0.03f;
    }

    namespace monster {
        inline constexpr float PathFollowingRefindDistance = 5.0f;
        inline constexpr float PathFollowingWaypointReachDistance = 0.1f;

        namespace melee {
            inline constexpr int Health = 100;
            inline constexpr float MoveSpeed = 3.0f;
            inline constexpr int AttackDamage = 10;
            inline constexpr float AttackRange = 2.0f;
            inline constexpr ecs::DeltaTime AttackCooldown{1.6f};
            inline constexpr ecs::DeltaTime AttackWindup{0.45f};
            inline constexpr ecs::DeltaTime AttackActive{0.1f};
            inline constexpr ecs::DeltaTime AttackRecovery{1.05f};
            inline constexpr float AttackMovementMultiplier = 0.0f;
        }

        namespace ranged {
            inline constexpr int Health = 35;
            inline constexpr float MoveSpeed = 3.5f;
            inline constexpr int AttackDamage = 12;
            inline constexpr float AttackRange = 10.5f;
            inline constexpr ecs::DeltaTime AttackCooldown{2.0f};
            inline constexpr ecs::DeltaTime AttackWindup{0.6f};
            inline constexpr ecs::DeltaTime AttackActive{0.05f};
            inline constexpr ecs::DeltaTime AttackRecovery{1.35f};
            inline constexpr float AttackMovementMultiplier = 0.0f;
            inline constexpr float ProjectileSpeed = 11.0f;
            inline constexpr float RetreatDistance = 7.0f;
        }
    }

    namespace blessing {
        namespace critical_strike {
            inline constexpr int BasePercent = 15;
            inline constexpr int PercentPerLevel = 4;
            inline constexpr int BaseDamagePercent = 175;
            inline constexpr int DamagePercentPerLevel = 15;
        }

        namespace life_steal {
            inline constexpr int BasePercent = 8;
            inline constexpr int PercentPerLevel = 2;
        }

        namespace burn_on_hit {
            inline constexpr int BaseDamagePerTick = 6;
            inline constexpr int DamagePerTickPerLevel = 2;
            inline constexpr ecs::DeltaTime BaseDuration{2.5f};
            inline constexpr ecs::DeltaTime DurationPerLevel{0.5f};
            inline constexpr ecs::DeltaTime TickInterval{1.0f};
        }

        namespace chain_lightning {
            inline constexpr int BaseDamagePercent = 50;
            inline constexpr int DamagePercentPerLevel = 10;
            inline constexpr int BaseSecondaryTargets = 1;
            inline constexpr int LevelsPerExtraTarget = 1;
            inline constexpr float JumpRadius = 9.0f;
        }

        namespace freeze_on_hit {
            inline constexpr int BasePercent = 15;
            inline constexpr int PercentPerLevel = 4;
            inline constexpr ecs::DeltaTime BaseDuration{1.0f};
            inline constexpr ecs::DeltaTime DurationPerLevel{0.15f};
        }
    }

    namespace progression {
        inline constexpr int BaseExperienceToNextLevel = 100;
        inline constexpr int ExperienceToNextLevelGrowth = 50;
        inline constexpr int MeleeMonsterExperience = 35;
        inline constexpr int RangedMonsterExperience = 45;
        inline constexpr std::size_t RewardOptionCount = 3;
        inline constexpr ecs::DeltaTime RewardSelectionTime{15.0f};
    }

    namespace spawn {
        inline constexpr std::size_t PlayerSlotCount = 4;
        inline constexpr float PlayerOffset = 2.0f;
        inline constexpr float MonsterRadius = 8.0f;
    }

    namespace room {
        inline constexpr float MinCoordinate = -20.0f;
        inline constexpr float MaxCoordinate = 20.0f;
        inline constexpr float PlayerSpawnY = -18.0f;
        inline constexpr float MonsterSpawnX = 8.0f;
        inline constexpr int CombatMeleeMonsterCount = 5;
        inline constexpr int CombatRangedMonsterCount = 2;
        inline constexpr int BossMeleeMonsterCount = 1;
    }

    namespace trap {
        namespace spikes {
            inline constexpr int Damage = 25;
            inline constexpr float CostMultiplier = 1.5f;
        }

        namespace poison_pool {
            inline constexpr int DamagePerTick = 6;
            inline constexpr ecs::DeltaTime Duration{3.0f};
            inline constexpr ecs::DeltaTime TickInterval{0.5f};
            inline constexpr float CostMultiplier = 1.8f;
        }

        namespace swamp {
            inline constexpr float MovementMultiplier = 0.35f;
            inline constexpr ecs::DeltaTime Duration{0.25f};
            inline constexpr float CostMultiplier = 1.2f;
        }
    }

}
