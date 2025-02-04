/*!
* Tabler v1.0.0 (https://tabler.io)
* @version 1.0.0
* @link https://tabler.io
* Copyright 2018-2025 The Tabler Authors
* Copyright 2018-2025 codecalm.net Paweł Kuna
* Licensed under MIT (https://github.com/tabler/tabler/blob/master/LICENSE)
*/
(function (factory) {
  typeof define === 'function' && define.amd ? define(factory) :
  factory();
})((function () { 'use strict';

  function _arrayLikeToArray(r, a) {
    (null == a || a > r.length) && (a = r.length);
    for (var e = 0, n = Array(a); e < a; e++) n[e] = r[e];
    return n;
  }
  function _arrayWithHoles(r) {
    if (Array.isArray(r)) return r;
  }
  function _iterableToArrayLimit(r, l) {
    var t = null == r ? null : "undefined" != typeof Symbol && r[Symbol.iterator] || r["@@iterator"];
    if (null != t) {
      var e,
        n,
        i,
        u,
        a = [],
        f = !0,
        o = !1;
      try {
        if (i = (t = t.call(r)).next, 0 === l) {
          if (Object(t) !== t) return;
          f = !1;
        } else for (; !(f = (e = i.call(t)).done) && (a.push(e.value), a.length !== l); f = !0);
      } catch (r) {
        o = !0, n = r;
      } finally {
        try {
          if (!f && null != t.return && (u = t.return(), Object(u) !== u)) return;
        } finally {
          if (o) throw n;
        }
      }
      return a;
    }
  }
  function _nonIterableRest() {
    throw new TypeError("Invalid attempt to destructure non-iterable instance.\nIn order to be iterable, non-array objects must have a [Symbol.iterator]() method.");
  }
  function _slicedToArray(r, e) {
    return _arrayWithHoles(r) || _iterableToArrayLimit(r, e) || _unsupportedIterableToArray(r, e) || _nonIterableRest();
  }
  function _unsupportedIterableToArray(r, a) {
    if (r) {
      if ("string" == typeof r) return _arrayLikeToArray(r, a);
      var t = {}.toString.call(r).slice(8, -1);
      return "Object" === t && r.constructor && (t = r.constructor.name), "Map" === t || "Set" === t ? Array.from(r) : "Arguments" === t || /^(?:Ui|I)nt(?:8|16|32)(?:Clamped)?Array$/.test(t) ? _arrayLikeToArray(r, a) : void 0;
    }
  }

  var items = {
    "menu-position": {
      localStorage: "tablerMenuPosition",
      default: "top"
    },
    "menu-behavior": {
      localStorage: "tablerMenuBehavior",
      default: "sticky"
    },
    "container-layout": {
      localStorage: "tablerContainerLayout",
      default: "boxed"
    }
  };
  var config = {};
  for (var _i = 0, _Object$entries = Object.entries(items); _i < _Object$entries.length; _i++) {
    var _Object$entries$_i = _slicedToArray(_Object$entries[_i], 2),
      key = _Object$entries$_i[0],
      params = _Object$entries$_i[1];
    var lsParams = localStorage.getItem(params.localStorage);
    config[key] = lsParams ? lsParams : params.default;
  }
  var parseUrl = function parseUrl() {
    var search = window.location.search.substring(1);
    var params = search.split("&");
    for (var i = 0; i < params.length; i++) {
      var arr = params[i].split("=");
      var _key = arr[0];
      var value = arr[1];
      if (!!items[_key]) {
        localStorage.setItem(items[_key].localStorage, value);
        config[_key] = value;
      }
    }
  };
  var toggleFormControls = function toggleFormControls(form) {
    for (var _i2 = 0, _Object$entries2 = Object.entries(items); _i2 < _Object$entries2.length; _i2++) {
      var _Object$entries2$_i = _slicedToArray(_Object$entries2[_i2], 2),
        _key2 = _Object$entries2$_i[0];
        _Object$entries2$_i[1];
      var elem = form.querySelector("[name=\"settings-".concat(_key2, "\"][value=\"").concat(config[_key2], "\"]"));
      if (elem) {
        elem.checked = true;
      }
    }
  };
  var submitForm = function submitForm(form) {
    for (var _i3 = 0, _Object$entries3 = Object.entries(items); _i3 < _Object$entries3.length; _i3++) {
      var _Object$entries3$_i = _slicedToArray(_Object$entries3[_i3], 2),
        _key3 = _Object$entries3$_i[0],
        _params2 = _Object$entries3$_i[1];
      var value = form.querySelector("[name=\"settings-".concat(_key3, "\"]:checked")).value;
      localStorage.setItem(_params2.localStorage, value);
      config[_key3] = value;
    }
    window.dispatchEvent(new Event("resize"));
    new bootstrap.Offcanvas(form).hide();
  };
  parseUrl();
  var form = document.querySelector("#offcanvasSettings");
  if (form) {
    form.addEventListener("submit", function (e) {
      e.preventDefault();
      submitForm(form);
    });
    toggleFormControls(form);
  }

}));
