export const openNavigationGroups = async (page) => {
  const navigation = page.getByRole("navigation", { name: "Navigasi utama" });
  const closedGroups = navigation.locator('button[aria-expanded="false"]:visible');

  while (await closedGroups.count()) {
    await closedGroups.first().click();
  }

  return navigation;
};
