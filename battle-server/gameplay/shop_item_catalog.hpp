#pragma once
#include <vector>


#include "shop_item.hpp"
namespace battle {
    std::vector<ShopItemDefinition> default_shop_item_definitions();

    std::vector<ShopOffer> default_shop_offers();
}
