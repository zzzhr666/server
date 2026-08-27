#pragma once
#include <vector>


#include "shop_item.hpp"
namespace battle {
    /// @brief 返回默认商店商品定义目录。
    std::vector<ShopItemDefinition> default_shop_item_definitions();

    /// @brief 返回包含全部阶段商品的默认商店报价。
    std::vector<ShopOffer> default_shop_offers();
    /// @brief 返回前期奖励房使用的商店报价。
    std::vector<ShopOffer> early_reward_shop_offers();
    /// @brief 返回后期奖励房使用的商店报价。
    std::vector<ShopOffer> late_reward_shop_offers();
}
