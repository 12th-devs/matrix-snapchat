import crypto from "node:crypto";

const SNAPCHAT_URL = "https://www.snapchat.com/web";
const unsupportedEventLabel = "[Unsupported Snapchat event]";

function normalizeText(value) {
  return String(value || "").replace(/\s+/g, " ").trim();
}


function prettifyChatText(value) {
  return normalizeText(
    value
      .replace(/Close Chat/gi, "")
      .replace(/·/g, " · ")
      .replace(/(Opened|Received|Sent|Replied|Reply|Say Hi!?|Typing)/g, " $1 ")
      .replace(/(\d+\s*[mhdwy])/gi, " $1 ")
      .replace(/(🔥)/g, " $1 ")
  );
}


function stableID(parts) {
  return crypto.createHash("sha1").update(parts.join("::")).digest("hex").slice(0, 16);
}


function readProtoVarint(buffer, offset) {
  let value = 0;
  let shift = 0;
  let cursor = offset;
  while (cursor < buffer.length && shift < 64) {
    const byte = buffer[cursor++];
    value |= (byte & 0x7f) << shift;
    if ((byte & 0x80) === 0) {
      return { value, offset: cursor };
    }
    shift += 7;
  }
  return null;
}


function readProtoBytesField(buffer, fieldNumber) {
  let offset = 0;
  while (offset < buffer.length) {
    const tag = readProtoVarint(buffer, offset);
    if (!tag) {
      return null;
    }
    offset = tag.offset;
    const currentField = tag.value >> 3;
    const wireType = tag.value & 7;
    if (wireType === 2) {
      const length = readProtoVarint(buffer, offset);
      if (!length) {
        return null;
      }
      offset = length.offset;
      if (offset + length.value > buffer.length) {
        return null;
      }
      const value = buffer.subarray(offset, offset + length.value);
      if (currentField === fieldNumber) {
        return value;
      }
      offset += length.value;
      continue;
    }
    if (wireType === 0) {
      const skipped = readProtoVarint(buffer, offset);
      if (!skipped) {
        return null;
      }
      offset = skipped.offset;
      continue;
    }
    if (wireType === 1) {
      offset += 8;
      continue;
    }
    if (wireType === 5) {
      offset += 4;
      continue;
    }
    return null;
  }
  return null;
}


function uuidFromBytes(bytes) {
  if (!bytes || bytes.length !== 16) {
    return "";
  }
  const hex = Buffer.from(bytes).toString("hex");
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}


function extractEncodedUUIDField(payload, fieldNumber) {
  const uuidMessage = readProtoBytesField(payload, fieldNumber);
  if (!uuidMessage) {
    return "";
  }
  return uuidFromBytes(readProtoBytesField(uuidMessage, 1));
}


function extractUUIDStringField(payload, fieldNumber) {
  const value = readProtoBytesField(payload, fieldNumber);
  if (!value) {
    return "";
  }
  const candidate = Buffer.from(value).toString("utf8").trim();
  return looksLikeConversationID(candidate) ? candidate.toLowerCase() : "";
}


function inspectProtoUUIDCandidates(payload, path = "", depth = 0, output = []) {
  if (!payload || depth > 5 || output.length >= 50) {
    return output;
  }
  let offset = 0;
  while (offset < payload.length && output.length < 50) {
    const tag = readProtoVarint(payload, offset);
    if (!tag || tag.value === 0) {
      break;
    }
    offset = tag.offset;
    const fieldNumber = tag.value >>> 3;
    const wireType = tag.value & 7;
    const fieldPath = path ? `${path}.${fieldNumber}` : String(fieldNumber);
    if (wireType === 2) {
      const length = readProtoVarint(payload, offset);
      if (!length || length.value < 0 || length.offset + length.value > payload.length) {
        break;
      }
      const value = payload.subarray(length.offset, length.offset + length.value);
      const text = Buffer.from(value).toString("utf8").trim();
      if (looksLikeConversationID(text)) {
        output.push({ path: fieldPath, kind: "string", uuid: text.toLowerCase() });
      } else if (value.length === 16) {
        output.push({ path: fieldPath, kind: "bytes", uuid: uuidFromBytes(value) });
      } else if (value.length >= 2) {
        inspectProtoUUIDCandidates(value, fieldPath, depth + 1, output);
      }
      offset = length.offset + length.value;
      continue;
    }
    if (wireType === 0) {
      const value = readProtoVarint(payload, offset);
      if (!value) {
        break;
      }
      offset = value.offset;
      continue;
    }
    if (wireType === 1) {
      offset += 8;
      continue;
    }
    if (wireType === 5) {
      offset += 4;
      continue;
    }
    break;
  }
  return output;
}


function extractSelfUserIDFromMessagingRequest(requestURL, postData) {
  if (!postData || postData.length < 6) {
    return "";
  }
  const payload = postData.subarray(5);
  if (/CircumstancesService\/targetingQuery$/i.test(requestURL)) {
    // Current Snapchat web COF requests call the account UUID `ghostId`
    // (field 3). Field 17 is retained as a fallback for deployments that
    // populate the newer `userId` member instead.
    return extractUUIDStringField(payload, 3) || extractUUIDStringField(payload, 17);
  }
  if (/\/(?:GetGroups|SyncConversations|QueryConversations)$/i.test(requestURL)) {
    return extractEncodedUUIDField(payload, 1);
  }
  if (/\/(?:DeltaSync|QueryMessages|UpdateContentMessage)$/i.test(requestURL)) {
    return extractEncodedUUIDField(payload, 4);
  }
  return "";
}


function makeChatID(name) {
  return stableID([normalizeText(name)]);
}


function looksLikeConversationID(value) {
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(String(value || "").trim());
}


function conversationURL(chatID) {
  return `${SNAPCHAT_URL}/${chatID}`;
}


function getConversationIDFromURL(value) {
  try {
    const url = new URL(value);
    const match = url.pathname.match(/^\/web\/([0-9a-f-]+)$/i);
    return match ? match[1] : "";
  } catch {
    return "";
  }
}


function inferAuthorFromEventText(text, fallbackName = "") {
  const normalized = normalizeText(text);
  if (!normalized) {
    return fallbackName;
  }
  const noPrefix = normalized.replace(/^\[Unsupported Snapchat event\]\s*/i, "");
  const youAndMatch = noPrefix.match(/^YOU AND (.+?) STARTED A SNAPSTREAK/i);
  if (youAndMatch) {
    return normalizeText(youAndMatch[1]) || fallbackName;
  }
  const triedCallMatch = noPrefix.match(/^(.+?) TRIED TO CALL YOU/i);
  if (triedCallMatch) {
    return normalizeText(triedCallMatch[1]) || fallbackName;
  }
  return fallbackName;
}


function inferEventAuthor(text, fallbackName = "") {
  const normalized = normalizeText(text);
  if (!normalized.startsWith(unsupportedEventLabel)) {
    return "";
  }
  if (/started a snapstreak|tried to call you/i.test(normalized)) {
    return "System";
  }
  return inferAuthorFromEventText(text, fallbackName);
}


export {
  normalizeText,
  prettifyChatText,
  stableID,
  extractSelfUserIDFromMessagingRequest,
  inspectProtoUUIDCandidates,
  makeChatID,
  looksLikeConversationID,
  conversationURL,
  getConversationIDFromURL,
  inferAuthorFromEventText,
  inferEventAuthor,
};
