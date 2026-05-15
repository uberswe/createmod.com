(function() {
  'use strict';

  if (!window.PublicKeyCredential) return;

  function bufferToBase64url(buf) {
    var bytes = new Uint8Array(buf);
    var str = '';
    for (var i = 0; i < bytes.length; i++) str += String.fromCharCode(bytes[i]);
    return btoa(str).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
  }

  function base64urlToBuffer(b64) {
    var str = b64.replace(/-/g, '+').replace(/_/g, '/');
    while (str.length % 4) str += '=';
    var raw = atob(str);
    var buf = new Uint8Array(raw.length);
    for (var i = 0; i < raw.length; i++) buf[i] = raw.charCodeAt(i);
    return buf.buffer;
  }

  function decodePublicKeyOptions(options) {
    if (options.challenge) {
      options.challenge = base64urlToBuffer(options.challenge);
    }
    if (options.user && options.user.id) {
      options.user.id = base64urlToBuffer(options.user.id);
    }
    if (options.excludeCredentials) {
      options.excludeCredentials = options.excludeCredentials.map(function(c) {
        c.id = base64urlToBuffer(c.id);
        return c;
      });
    }
    if (options.allowCredentials) {
      options.allowCredentials = options.allowCredentials.map(function(c) {
        c.id = base64urlToBuffer(c.id);
        return c;
      });
    }
    return options;
  }

  window.passkeyRegister = function(nameInput) {
    var name = '';
    if (nameInput) name = nameInput.value || '';

    fetch('/settings/security/passkeys/register/begin', { method: 'POST', credentials: 'same-origin' })
      .then(function(r) { return r.json(); })
      .then(function(options) {
        var publicKey = decodePublicKeyOptions(options.publicKey);
        return navigator.credentials.create({ publicKey: publicKey });
      })
      .then(function(credential) {
        var response = credential.response;
        var body = JSON.stringify({
          id: credential.id,
          rawId: bufferToBase64url(credential.rawId),
          type: credential.type,
          response: {
            attestationObject: bufferToBase64url(response.attestationObject),
            clientDataJSON: bufferToBase64url(response.clientDataJSON)
          }
        });
        var url = '/settings/security/passkeys/register/finish';
        if (name) url += '?name=' + encodeURIComponent(name);
        return fetch(url, {
          method: 'POST',
          credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json' },
          body: body
        });
      })
      .then(function(r) { return r.json(); })
      .then(function(result) {
        if (result.status === 'ok') {
          window.location.reload();
        } else {
          alert(result.error || 'Registration failed');
        }
      })
      .catch(function(err) {
        if (err.name !== 'NotAllowedError') {
          alert('Passkey registration failed: ' + err.message);
        }
      });
  };

  window.passkeyLogin = function() {
    fetch('/auth/passkey/begin', { method: 'POST', credentials: 'same-origin' })
      .then(function(r) { return r.json(); })
      .then(function(options) {
        var publicKey = decodePublicKeyOptions(options.publicKey);
        return navigator.credentials.get({ publicKey: publicKey });
      })
      .then(function(credential) {
        var response = credential.response;
        var body = JSON.stringify({
          id: credential.id,
          rawId: bufferToBase64url(credential.rawId),
          type: credential.type,
          response: {
            authenticatorData: bufferToBase64url(response.authenticatorData),
            clientDataJSON: bufferToBase64url(response.clientDataJSON),
            signature: bufferToBase64url(response.signature),
            userHandle: response.userHandle ? bufferToBase64url(response.userHandle) : ''
          }
        });
        return fetch('/auth/passkey/finish', {
          method: 'POST',
          credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json' },
          body: body
        });
      })
      .then(function(r) { return r.json(); })
      .then(function(result) {
        if (result.status === 'ok') {
          window.location.href = result.redirect || '/';
        } else {
          alert(result.error || 'Login failed');
        }
      })
      .catch(function(err) {
        if (err.name !== 'NotAllowedError') {
          alert('Passkey login failed: ' + err.message);
        }
      });
  };

  document.querySelectorAll('[data-passkey-register]').forEach(function(btn) {
    btn.style.display = '';
    btn.addEventListener('click', function() {
      var nameInput = document.getElementById('passkey-name');
      window.passkeyRegister(nameInput);
    });
  });

  document.querySelectorAll('[data-passkey-login]').forEach(function(btn) {
    btn.style.display = '';
    btn.addEventListener('click', function() {
      window.passkeyLogin();
    });
  });
})();
