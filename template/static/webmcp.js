// WebMCP tools for CreateMod.com.
//
// Registers site tools with the browser's model context API so AI agents can
// act on the user's behalf. Two API surfaces exist in the wild:
//   - navigator.modelContext.provideContext({tools}) (Chrome early preview)
//   - document.modelContext.registerTool(tool) (W3C WebMCP draft)
// Both are supported; whichever exists wins.
//
// Site policy baked into every tool: agents never download schematic (.nbt)
// files. Users are sent to schematic pages or generator pages (with shareable
// configuration links) to download builds themselves.
//
// Pages can contribute page-specific tools (e.g. generators) by pushing tool
// definitions to window.CreateModAgentTools before or after this script runs.
(function () {
  'use strict';
  if (window._cmWebMCPInit) return;
  window._cmWebMCPInit = true;

  var registered = [];

  function textResult(text) {
    return { content: [{ type: 'text', text: text }] };
  }

  function modelContext() {
    return (typeof navigator !== 'undefined' && navigator.modelContext) ||
           (typeof document !== 'undefined' && document.modelContext) || null;
  }

  function syncTools() {
    var mc = modelContext();
    if (!mc) return;
    try {
      if (typeof mc.provideContext === 'function') {
        // provideContext replaces the full tool set; always pass everything.
        mc.provideContext({ tools: registered.slice() });
      } else if (typeof mc.registerTool === 'function') {
        while (syncTools._next < registered.length) {
          mc.registerTool(registered[syncTools._next]);
          syncTools._next++;
        }
      }
    } catch (e) { /* agent API unavailable or rejected registration */ }
  }
  syncTools._next = 0;

  function addTools(tools) {
    for (var i = 0; i < tools.length; i++) registered.push(tools[i]);
    syncTools();
  }

  var NBT_POLICY = ' Never download schematic (.nbt) files; send the user to the page instead.';

  var coreTools = [
    {
      name: 'search-schematics',
      description: 'Search CreateMod.com for Minecraft Create Mod schematics. Returns matching builds with their page URLs.' + NBT_POLICY,
      inputSchema: {
        type: 'object',
        properties: {
          query: { type: 'string', description: 'Search terms, e.g. "elevator" or "wheat farm"' }
        },
        required: ['query']
      },
      annotations: { readOnlyHint: true },
      execute: function (input) {
        var q = (input && input.query) || '';
        return fetch('/api/search/suggest?q=' + encodeURIComponent(q))
          .then(function (r) { return r.json(); })
          .then(function (suggestions) {
            if (!suggestions || !suggestions.length) {
              return textResult('No matches for "' + q + '". Try broader terms, or open /search?q=' + encodeURIComponent(q));
            }
            var lines = suggestions.map(function (s) {
              var slug = s.slug || s.Slug || s.name || s.Name || '';
              var title = s.title || s.Title || slug;
              return '- ' + title + ' — https://createmod.com/schematics/' + slug;
            });
            return textResult('Matches for "' + q + '":\n' + lines.join('\n'));
          });
      }
    },
    {
      name: 'get-schematic-details',
      description: 'Get curated details for one schematic by its URL slug: description, author, rating, versions, required mods, video and material list.' + NBT_POLICY,
      inputSchema: {
        type: 'object',
        properties: {
          name: { type: 'string', description: 'The schematic URL slug, e.g. "easy-helicopter-survival-friendly"' }
        },
        required: ['name']
      },
      annotations: { readOnlyHint: true },
      execute: function (input) {
        var name = (input && input.name) || '';
        return fetch('/schematics/' + encodeURIComponent(name), { headers: { Accept: 'text/markdown' } })
          .then(function (r) {
            if (!r.ok) throw new Error('not found');
            return r.text();
          })
          .then(function (md) { return textResult(md); })
          .catch(function () { return textResult('Schematic "' + name + '" not found.'); });
      }
    },
    {
      name: 'open-schematic-page',
      description: 'Navigate the user to a schematic page on CreateMod.com. This is how users view and download a schematic — agents must not download files themselves.',
      inputSchema: {
        type: 'object',
        properties: {
          name: { type: 'string', description: 'The schematic URL slug' }
        },
        required: ['name']
      },
      execute: function (input) {
        var name = (input && input.name) || '';
        var url = '/schematics/' + encodeURIComponent(name);
        window.location.assign(url);
        return Promise.resolve(textResult('Navigating to https://createmod.com' + url));
      }
    },
    {
      name: 'open-generator',
      description: 'Navigate the user to one of the CreateMod.com structure generators (ship hull, airship balloon, or propeller). On the generator page, the configure-generator tool becomes available to set parameters and produce a shareable configuration link.' + NBT_POLICY,
      inputSchema: {
        type: 'object',
        properties: {
          generator: { type: 'string', enum: ['hull', 'balloon', 'propeller'], description: 'Which generator to open' },
          config: { type: 'string', description: 'Optional encoded configuration from a previously shared generator link' }
        },
        required: ['generator']
      },
      execute: function (input) {
        var gen = (input && input.generator) || 'hull';
        if (['hull', 'balloon', 'propeller'].indexOf(gen) === -1) gen = 'hull';
        var url = '/generators/' + gen;
        if (input && input.config && /^[A-Za-z0-9_-]+$/.test(input.config)) {
          url += '/' + input.config;
        }
        window.location.assign(url);
        return Promise.resolve(textResult('Navigating to https://createmod.com' + url));
      }
    }
  ];

  addTools(coreTools);

  // Page-specific tools: pages push definitions to window.CreateModAgentTools.
  var pending = window.CreateModAgentTools;
  window.CreateModAgentTools = {
    push: function (tool) { addTools([tool]); }
  };
  if (pending && pending.length) addTools(pending);

  // The model context API may attach after us; retry briefly.
  var attempts = 0;
  var poll = setInterval(function () {
    attempts++;
    if (modelContext()) { syncTools(); clearInterval(poll); }
    else if (attempts > 20) { clearInterval(poll); }
  }, 500);
})();
