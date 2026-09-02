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

  // canSend and sendDisabledReason answer the same question from two sides;
  // if they ever disagree the UI shows a live box with a reason under it.
  test("a reason is given exactly when sending is closed", () => {
    STATUSES.forEach((status) => {
      ROLES.forEach((role) => {
        const view = deriveThreadView(makeConversation({ status, role }));
        expect(view.canSend).toBe(view.sendDisabledReason === null);
      });
    });
  });

  test("each side is told why in its own words", () => {
    expect(
      deriveThreadView(makeConversation({ status: "pending", role: "buyer" })).sendDisabledReason,
    ).toBe("Waiting for the seller to accept your request.");
    expect(
      deriveThreadView(makeConversation({ status: "pending", role: "seller" })).sendDisabledReason,
    ).toBe("Accept this request to reply.");
    expect(
      deriveThreadView(makeConversation({ status: "declined", role: "buyer" })).sendDisabledReason,
    ).toBe("The seller declined this request.");
    expect(
      deriveThreadView(makeConversation({ status: "declined", role: "seller" })).sendDisabledReason,
    ).toBe("You declined this request.");
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
      expect(deriveThreadView(makeConversation({ status })).statusLabel).toBeTruthy();
    });
  });
});
