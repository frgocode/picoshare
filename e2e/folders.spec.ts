import { test, expect } from "./fixtures";
import { login } from "./helpers/login";

test.describe.configure({ mode: "parallel" });

test("creates folders, navigates with breadcrumbs, and uploads into a folder", async ({
  page,
}) => {
  await login(page);

  await page.getByRole("menuitem", { name: "Files" }).click();
  await expect(page).toHaveURL("/files");

  await page.locator("#folder-name").fill("Projects");
  await page.getByRole("button", { name: "Create folder" }).click();
  await expect(
    page.getByRole("row").filter({ hasText: "Projects" }),
  ).toBeVisible();

  await page.getByRole("link", { name: "Projects" }).click();
  await expect(page).toHaveURL(/\/files\?folder=.+/);

  await page.locator("#folder-name").fill("Sub");
  await page.getByRole("button", { name: "Create folder" }).click();
  await expect(page.getByRole("row").filter({ hasText: "Sub" })).toBeVisible();

  await page.getByRole("link", { name: "Sub" }).click();
  await expect(page).toHaveURL(/\/files\?folder=.+/);

  await expect(page.getByRole("link", { name: "Root" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Projects" })).toBeVisible();

  await page
    .getByRole("link", { name: "Upload a file to this folder" })
    .click();
  await expect(page).toHaveURL(/\/?\?folder=.+/);

  await page.locator(".file-input").setInputFiles([
    {
      name: "folder-upload.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("contents in subfolder"),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!",
  );

  await page.getByRole("menuitem", { name: "Files" }).click();
  await expect(page).toHaveURL("/files");
  await expect(
    page.getByRole("row").filter({ hasText: "folder-upload.txt" }),
  ).toHaveCount(0);
});

test("moves a file between folders and back to root", async ({ page }) => {
  await login(page);

  await page.getByRole("menuitem", { name: "Files" }).click();
  await page.locator("#folder-name").fill("MoveTarget");
  await page.getByRole("button", { name: "Create folder" }).click();
  await expect(
    page.getByRole("row").filter({ hasText: "MoveTarget" }),
  ).toBeVisible();

  await page.getByRole("menuitem", { name: "Upload" }).click();
  await page.locator(".file-input").setInputFiles([
    {
      name: "movable.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("movable contents"),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!",
  );

  await page.getByRole("menuitem", { name: "Files" }).click();
  await page
    .getByRole("row")
    .filter({ hasText: "movable.txt" })
    .getByRole("link", { name: "Move" })
    .click();
  await expect(page).toHaveURL(/\/files\/.+\/move$/);
  await page
    .locator("#destination-folder")
    .selectOption({ label: "MoveTarget" });
  await page.getByRole("button", { name: "Move" }).click();
  await expect(page).toHaveURL("/files");

  await expect(
    page.getByRole("row").filter({ hasText: "movable.txt" }),
  ).toHaveCount(0);

  await page.getByRole("link", { name: "MoveTarget" }).click();
  await expect(
    page.getByRole("row").filter({ hasText: "movable.txt" }),
  ).toBeVisible();

  await page
    .getByRole("row")
    .filter({ hasText: "movable.txt" })
    .getByRole("link", { name: "movable.txt" })
    .click();
  await expect(page.locator("pre")).toHaveText("movable contents");

  await page.goBack();
  await page
    .getByRole("row")
    .filter({ hasText: "movable.txt" })
    .getByRole("link", { name: "Move" })
    .click();
  await page.locator("#destination-folder").selectOption({ label: "Root" });
  await page.getByRole("button", { name: "Move" }).click();
  await expect(page).toHaveURL("/files");

  await page.getByRole("link", { name: "MoveTarget" }).click();
  await expect(
    page.getByRole("row").filter({ hasText: "movable.txt" }),
  ).toHaveCount(0);
});

test("renames a folder and rejects duplicate names", async ({ page }) => {
  await login(page);

  await page.getByRole("menuitem", { name: "Files" }).click();
  await page.locator("#folder-name").fill("OriginalName");
  await page.getByRole("button", { name: "Create folder" }).click();
  await expect(
    page.getByRole("row").filter({ hasText: "OriginalName" }),
  ).toBeVisible();

  await page.locator("#folder-name").fill("OtherName");
  await page.getByRole("button", { name: "Create folder" }).click();
  await expect(
    page.getByRole("row").filter({ hasText: "OtherName" }),
  ).toBeVisible();

  await page
    .getByRole("row")
    .filter({ hasText: "OriginalName" })
    .getByRole("link", { name: /Rename folder/ })
    .click();
  await expect(page).toHaveURL(/\/folders\/.+\/edit$/);
  await page.locator("#folder-name").fill("RenamedFolder");
  await page.getByRole("button", { name: "Save" }).click();
  await expect(page).toHaveURL("/files");
  await expect(
    page.getByRole("row").filter({ hasText: "RenamedFolder" }),
  ).toBeVisible();

  await page
    .getByRole("row")
    .filter({ hasText: "RenamedFolder" })
    .getByRole("link", { name: /Rename folder/ })
    .click();
  await page.locator("#folder-name").fill("OtherName");
  await page.getByRole("button", { name: "Save" }).click();
  await expect(page.locator("#error-message")).toBeVisible();
});

test("moves a folder and rejects circular moves", async ({ page }) => {
  await login(page);

  await page.getByRole("menuitem", { name: "Files" }).click();
  await page.locator("#folder-name").fill("ParentFolder");
  await page.getByRole("button", { name: "Create folder" }).click();
  await page.locator("#folder-name").fill("SiblingFolder");
  await page.getByRole("button", { name: "Create folder" }).click();

  await page.getByRole("link", { name: "ParentFolder" }).click();
  await page.locator("#folder-name").fill("ChildFolder");
  await page.getByRole("button", { name: "Create folder" }).click();
  await expect(
    page.getByRole("row").filter({ hasText: "ChildFolder" }),
  ).toBeVisible();

  await page.getByRole("link", { name: "Root" }).click();
  await expect(page).toHaveURL("/files");

  await page
    .getByRole("row")
    .filter({ hasText: "ChildFolder" })
    .count()
    .then((count) => expect(count).toBe(0));

  await page
    .getByRole("row")
    .filter({ hasText: "ParentFolder" })
    .getByRole("link", { name: /Move folder/ })
    .click();
  await page.locator("#parent-folder").selectOption({ label: "SiblingFolder" });
  await page.getByRole("button", { name: "Move" }).click();
  await expect(page).toHaveURL("/files");

  await page.getByRole("link", { name: "SiblingFolder" }).click();
  await expect(
    page.getByRole("row").filter({ hasText: "ParentFolder" }),
  ).toBeVisible();
});

test("deletes an empty folder and rejects deletion of a nonempty folder", async ({
  page,
}) => {
  await login(page);

  await page.getByRole("menuitem", { name: "Files" }).click();
  await page.locator("#folder-name").fill("EmptyFolder");
  await page.getByRole("button", { name: "Create folder" }).click();
  await page.locator("#folder-name").fill("FullFolder");
  await page.getByRole("button", { name: "Create folder" }).click();

  await page.getByRole("link", { name: "FullFolder" }).click();
  await page.locator("#folder-name").fill("InnerFolder");
  await page.getByRole("button", { name: "Create folder" }).click();
  await page.getByRole("link", { name: "Root" }).click();

  await page
    .getByRole("row")
    .filter({ hasText: "EmptyFolder" })
    .getByRole("link", { name: /Delete folder/ })
    .click();
  await expect(page).toHaveURL(/\/folders\/.+\/confirm-delete$/);
  await page.getByRole("button", { name: "Delete" }).click();
  await expect(page).toHaveURL("/files");
  await expect(
    page.getByRole("row").filter({ hasText: "EmptyFolder" }),
  ).toHaveCount(0);

  await page
    .getByRole("row")
    .filter({ hasText: "FullFolder" })
    .getByRole("link", { name: /Delete folder/ })
    .click();
  await page.getByRole("button", { name: "Delete" }).click();
  await expect(page.locator("#error-message")).toBeVisible();
});
