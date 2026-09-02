// Mirrors backend/internal/service/conversation.go. If they disagree, this is wrong.
import { deriveThreadView } from "../lib/chatState";
import { makeConversation } from "../test/factories";
import type { ConversationRole, ConversationStatus } from "../api/types";

const ROLES: ConversationRole[] = ["buyer", "seller"];
const STATUSES: ConversationStatus[] = ["pending", "accepted", "declined"];

describe("deriveThreadView - sending", () => {
  test("only an accepted thread takes messages", () => {
    ROLES.forEach((role) => {
      expect(deriveThreadView(makeConversation({ status: "accepted", role })).canSend).toBe(true);
      expect(deriveThreadView(makeConversation({ status: "pending", role })).canSend).toBe(false);
      expect(deriveThreadView(makeConversation({ status: "declined", role })).canSend).toBe(false);
    });
  });

  // canSend and sendDisabledKey answer the same question from two sides;
  // if they ever disagree the UI shows a live box with a reason under it.
  test("a reason is given exactly when sending is closed", () => {
    STATUSES.forEach((status) => {
      ROLES.forEach((role) => {
        const view = deriveThreadView(makeConversation({ status, role }));
        expect(view.canSend).toBe(view.sendDisabledKey === null);
      });
    });
  });

  test("each side gets its own key", () => {
    expect(
      deriveThreadView(makeConversation({ status: "pending", role: "buyer" })).sendDisabledKey,
    ).toBe("chat.disabled.waitingSeller");
    expect(
      deriveThreadView(makeConversation({ status: "pending", role: "seller" })).sendDisabledKey,
    ).toBe("chat.disabled.sellerAccept");
    expect(
      deriveThreadView(makeConversation({ status: "declined", role: "buyer" })).sendDisabledKey,
    ).toBe("chat.disabled.sellerDeclined");
    expect(
      deriveThreadView(makeConversation({ status: "declined", role: "seller" })).sendDisabledKey,
    ).toBe("chat.disabled.youDeclined");
  });
});

describe("deriveThreadView - deciding", () => {
  test("only the seller, and only while pending", () => {
    expect(
      deriveThreadView(makeConversation({ status: "pending", role: "seller" })).canDecide,
    ).toBe(true);
    expect(deriveThreadView(makeConversation({ status: "pending", role: "buyer" })).canDecide).toBe(
      false,
    );
  });

  test("an answered request can't be answered again", () => {
    expect(
      deriveThreadView(makeConversation({ status: "accepted", role: "seller" })).canDecide,
    ).toBe(false);
    expect(
      deriveThreadView(makeConversation({ status: "declined", role: "seller" })).canDecide,
    ).toBe(false);
  });
});

describe("deriveThreadView - status label", () => {
  test("every status has one", () => {
    STATUSES.forEach((status) => {
      expect(deriveThreadView(makeConversation({ status })).statusKey).toBeTruthy();
    });
  });
});
