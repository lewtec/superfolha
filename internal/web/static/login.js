(function () {
  var DB = "superfolha-login";
  var STORE = "keys";
  var KEY = "identity";

  function $(id) {
    return document.getElementById(id);
  }

  function setStatus(text, isError) {
    var el = $("sf-login-status");
    if (!el) return;
    el.textContent = text;
    el.classList.toggle("text-error", !!isError);
  }

  function openDB() {
    return new Promise(function (resolve, reject) {
      var req = indexedDB.open(DB, 1);
      req.onupgradeneeded = function () {
        req.result.createObjectStore(STORE);
      };
      req.onsuccess = function () {
        resolve(req.result);
      };
      req.onerror = function () {
        reject(req.error);
      };
    });
  }

  function idbGet(db) {
    return new Promise(function (resolve, reject) {
      var tx = db.transaction(STORE, "readonly");
      var req = tx.objectStore(STORE).get(KEY);
      req.onsuccess = function () {
        resolve(req.result || null);
      };
      req.onerror = function () {
        reject(req.error);
      };
    });
  }

  function idbPut(db, value) {
    return new Promise(function (resolve, reject) {
      var tx = db.transaction(STORE, "readwrite");
      var req = tx.objectStore(STORE).put(value, KEY);
      req.onsuccess = function () {
        resolve();
      };
      req.onerror = function () {
        reject(req.error);
      };
    });
  }

  function b64url(buf) {
    var bytes = new Uint8Array(buf);
    var bin = "";
    for (var i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i]);
    return btoa(bin).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
  }

  async function getKeyPair() {
    if (!window.crypto || !crypto.subtle || typeof crypto.subtle.generateKey !== "function") {
      throw new Error("missing WebCrypto");
    }
    var db = await openDB();
    var existing = await idbGet(db);
    if (existing && existing.privateKey && existing.publicKey) {
      return existing;
    }
    var pair = await crypto.subtle.generateKey({ name: "Ed25519" }, false, ["sign", "verify"]);
    await idbPut(db, pair);
    return pair;
  }

  async function signIn() {
    var root = $("sf-login");
    if (!root) return;
    var challengeURL = root.getAttribute("data-challenge");
    var verifyURL = root.getAttribute("data-verify");
    var next = root.getAttribute("data-next") || "";
    setStatus(root.getAttribute("data-working") || "Signing in…", false);
    try {
      var pair = await getKeyPair();
      var chRes = await fetch(challengeURL, { credentials: "same-origin" });
      if (!chRes.ok) throw new Error("challenge " + chRes.status);
      var chJSON = await chRes.json();
      var challenge = chJSON.challenge;
      var sig = await crypto.subtle.sign("Ed25519", pair.privateKey, new TextEncoder().encode(challenge));
      var pub = await crypto.subtle.exportKey("raw", pair.publicKey);
      var res = await fetch(verifyURL, {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          challenge: challenge,
          public_key: b64url(pub),
          signature: b64url(sig),
          next: next,
        }),
      });
      if (!res.ok) throw new Error("verify " + res.status);
      var out = await res.json();
      window.location.assign(out.next || "/sessions");
    } catch (err) {
      var unsupported = String(err && err.message || err).indexOf("Ed25519") !== -1 ||
        String(err).indexOf("not supported") !== -1 ||
        String(err).indexOf("missing WebCrypto") !== -1;
      setStatus(
        unsupported
          ? (root.getAttribute("data-unsupported") || "This browser cannot do Ed25519 login.")
          : (root.getAttribute("data-failed") || "Sign-in failed."),
        true
      );
    }
  }

  document.addEventListener("DOMContentLoaded", function () {
    var btn = $("sf-login-btn");
    if (!btn) return;
    btn.addEventListener("click", function (ev) {
      ev.preventDefault();
      signIn();
    });
  });
})();
