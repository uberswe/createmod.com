package pages

import (
	"createmod/internal/server"
	"net/http"
)

const openAPISpec = `{
  "openapi": "3.0.3",
  "info": {
    "title": "CreateMod.com API",
    "description": "Search, retrieve, upload, and analyze schematics on CreateMod.com.",
    "version": "1.0.0",
    "contact": { "email": "hello@createmod.com" },
    "termsOfService": "https://createmod.com/terms-of-service"
  },
  "servers": [
    { "url": "https://createmod.com", "description": "Production" }
  ],
  "security": [
    { "ApiKeyHeader": [] }
  ],
  "paths": {
    "/api/schematics": {
      "get": {
        "operationId": "listSchematics",
        "summary": "List or search schematics",
        "description": "Returns schematics filtered and sorted by the given parameters. With no query and no sort it browses by trending; with a query it sorts by relevance. Supports both API key and HMAC authentication.",
        "tags": ["Schematics"],
        "security": [
          { "ApiKeyHeader": [] },
          { "ApiKeyQuery": [] },
          { "HMACSignature": [] }
        ],
        "parameters": [
          { "name": "query", "in": "query", "schema": { "type": "string" }, "description": "Search term (alias: q)" },
          { "name": "page", "in": "query", "schema": { "type": "integer", "default": 1, "minimum": 1 }, "description": "Page number (alias: p)" },
          { "name": "per_page", "in": "query", "schema": { "type": "integer", "enum": [8, 16, 24, 32, 64, 100], "default": 24 }, "description": "Results per page" },
          { "name": "sort", "in": "query", "schema": { "type": "integer", "enum": [1, 2, 3, 4, 5, 6, 7, 8], "default": 1 }, "description": "Sort order: 1 best match, 2 newest, 3 oldest, 4 highest rated, 5 lowest rated, 6 most viewed, 7 least viewed, 8 trending" },
          { "name": "category", "in": "query", "schema": { "type": "string" }, "description": "Filter by category key (default: all)" },
          { "name": "mcv", "in": "query", "schema": { "type": "string" }, "description": "Filter by Minecraft version" },
          { "name": "cv", "in": "query", "schema": { "type": "string" }, "description": "Filter by Create version (or a ~major group like ~6.0)" },
          { "name": "rating", "in": "query", "schema": { "type": "integer", "minimum": 0, "maximum": 5 }, "description": "Minimum rating" },
          { "name": "tag", "in": "query", "schema": { "type": "string" }, "description": "Filter by tag key(s), comma-separated" },
          { "name": "mod", "in": "query", "schema": { "type": "string" }, "description": "Filter by required mod namespace; repeatable (or comma-separated via mods)" }
        ],
        "responses": {
          "200": {
            "description": "Paginated list of schematics",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/SchematicListResponse" }
              }
            }
          },
          "401": { "$ref": "#/components/responses/Unauthorized" },
          "429": { "$ref": "#/components/responses/RateLimited" }
        }
      }
    },
    "/api/schematics/{name}": {
      "get": {
        "operationId": "getSchematic",
        "summary": "Get schematic details",
        "description": "Returns detailed information about a single schematic. Supports both API key and HMAC authentication.",
        "tags": ["Schematics"],
        "security": [
          { "ApiKeyHeader": [] },
          { "ApiKeyQuery": [] },
          { "HMACSignature": [] }
        ],
        "parameters": [
          { "name": "name", "in": "path", "required": true, "schema": { "type": "string" }, "description": "The URL slug of the schematic" }
        ],
        "responses": {
          "200": {
            "description": "Schematic details",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/Schematic" }
              }
            }
          },
          "401": { "$ref": "#/components/responses/Unauthorized" },
          "404": { "$ref": "#/components/responses/NotFound" },
          "429": { "$ref": "#/components/responses/RateLimited" }
        }
      }
    },
    "/api/home": {
      "get": {
        "operationId": "getHome",
        "summary": "Home page rails",
        "description": "Returns the trending, latest, and highest rated schematic rails shown on the home page. Supports both API key and HMAC authentication.",
        "tags": ["Schematics"],
        "security": [
          { "ApiKeyHeader": [] },
          { "ApiKeyQuery": [] },
          { "HMACSignature": [] }
        ],
        "responses": {
          "200": {
            "description": "Home rails",
            "content": { "application/json": { "schema": { "$ref": "#/components/schemas/HomeResponse" } } }
          },
          "401": { "$ref": "#/components/responses/Unauthorized" },
          "429": { "$ref": "#/components/responses/RateLimited" }
        }
      }
    },
    "/api/schematics/filters": {
      "get": {
        "operationId": "getFilters",
        "summary": "Search filter options",
        "description": "Returns the category, Minecraft version, Create version, tag, and mod option lists used by the search filters. Supports both API key and HMAC authentication.",
        "tags": ["Schematics"],
        "security": [
          { "ApiKeyHeader": [] },
          { "ApiKeyQuery": [] },
          { "HMACSignature": [] }
        ],
        "responses": {
          "200": {
            "description": "Filter option lists",
            "content": { "application/json": { "schema": { "$ref": "#/components/schemas/FiltersResponse" } } }
          },
          "401": { "$ref": "#/components/responses/Unauthorized" },
          "429": { "$ref": "#/components/responses/RateLimited" }
        }
      }
    },
    "/api/schematics/changes": {
      "get": {
        "operationId": "getSchematicChanges",
        "summary": "List changed schematics",
        "description": "Returns schematics edited or removed after the given cursor, so external caches can invalidate precisely. Call without a cursor to get the current cursor (with an empty list) and initialise from now. Public (no API key).",
        "tags": ["Schematics"],
        "security": [],
        "parameters": [
          { "name": "cursor", "in": "query", "schema": { "type": "string" }, "description": "Opaque cursor token from a previous response. Omit to get the current cursor only." }
        ],
        "responses": {
          "200": {
            "description": "Changed schematics after the cursor",
            "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ChangesResponse" } } }
          },
          "400": { "$ref": "#/components/responses/BadRequest" },
          "429": { "$ref": "#/components/responses/RateLimited" }
        }
      }
    },
    "/api/schematics/stats": {
      "get": {
        "operationId": "getSchematicBulkStats",
        "summary": "Bulk schematic counters",
        "description": "Returns just the volatile counters (views, downloads, rating, comment count) for up to 100 schematics, so caches can keep content long-lived while refreshing counters often. Public.",
        "tags": ["Schematics"],
        "security": [],
        "parameters": [
          { "name": "names", "in": "query", "required": true, "schema": { "type": "string" }, "description": "Comma-separated schematic slugs (max 100)" }
        ],
        "responses": {
          "200": {
            "description": "Counters for the requested schematics",
            "content": { "application/json": { "schema": { "type": "array", "items": { "$ref": "#/components/schemas/StatItem" } } } }
          },
          "400": { "$ref": "#/components/responses/BadRequest" },
          "429": { "$ref": "#/components/responses/RateLimited" }
        }
      }
    },
    "/api/schematics/{name}/download": {
      "get": {
        "operationId": "downloadSchematic",
        "summary": "Download a schematic file",
        "description": "Counts the download and redirects to the schematic's .nbt file. Pass ?f={fileID} to download a variation file. Also available at the alias GET /api/download/{name}. Supports both API key and HMAC authentication.",
        "tags": ["Schematics"],
        "security": [
          { "ApiKeyHeader": [] },
          { "ApiKeyQuery": [] },
          { "HMACSignature": [] }
        ],
        "parameters": [
          { "name": "name", "in": "path", "required": true, "schema": { "type": "string" }, "description": "The URL slug of the schematic" },
          { "name": "f", "in": "query", "schema": { "type": "string" }, "description": "Optional variation file ID" }
        ],
        "responses": {
          "302": { "description": "Redirect to the schematic file" },
          "401": { "$ref": "#/components/responses/Unauthorized" },
          "404": { "$ref": "#/components/responses/NotFound" },
          "429": { "$ref": "#/components/responses/RateLimited" }
        }
      }
    },
    "/api/schematics/{name}/comments": {
      "get": {
        "operationId": "getSchematicComments",
        "summary": "List schematic comments",
        "description": "Returns the approved comment thread for a schematic. Supports both API key and HMAC authentication.",
        "tags": ["Schematics"],
        "security": [
          { "ApiKeyHeader": [] },
          { "ApiKeyQuery": [] },
          { "HMACSignature": [] }
        ],
        "parameters": [
          { "name": "name", "in": "path", "required": true, "schema": { "type": "string" }, "description": "The URL slug of the schematic" }
        ],
        "responses": {
          "200": {
            "description": "Comment thread",
            "content": { "application/json": { "schema": { "$ref": "#/components/schemas/CommentsResponse" } } }
          },
          "401": { "$ref": "#/components/responses/Unauthorized" },
          "404": { "$ref": "#/components/responses/NotFound" },
          "429": { "$ref": "#/components/responses/RateLimited" }
        }
      }
    },
    "/api/schematics/upload": {
      "post": {
        "operationId": "uploadSchematic",
        "summary": "Upload a .nbt schematic file",
        "description": "Upload an NBT file. Returns a preview token. The schematic is not published until you complete the publish flow. Supports both API key and HMAC authentication.",
        "tags": ["Schematics"],
        "security": [
          { "ApiKeyHeader": [] },
          { "ApiKeyQuery": [] },
          { "HMACSignature": [] }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "multipart/form-data": {
              "schema": {
                "type": "object",
                "required": ["file"],
                "properties": {
                  "file": { "type": "string", "format": "binary", "description": "The .nbt schematic file (max 10 MB)" }
                }
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "Upload result with preview token",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/UploadResponse" }
              }
            }
          },
          "400": { "$ref": "#/components/responses/BadRequest" },
          "401": { "$ref": "#/components/responses/Unauthorized" },
          "409": {
            "description": "Duplicate upload detected",
            "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Error" } } }
          },
          "429": { "$ref": "#/components/responses/RateLimited" }
        }
      }
    },
    "/api/schematics/{name}/stats": {
      "get": {
        "operationId": "getSchematicStats",
        "summary": "Get schematic analytics",
        "description": "Returns hourly analytics for the last 30 days. Only the schematic author can access this endpoint. API key authentication only (HMAC not supported).",
        "tags": ["Analytics"],
        "security": [
          { "ApiKeyHeader": [] },
          { "ApiKeyQuery": [] }
        ],
        "parameters": [
          { "name": "name", "in": "path", "required": true, "schema": { "type": "string" }, "description": "The URL slug of the schematic" }
        ],
        "responses": {
          "200": {
            "description": "Schematic analytics data",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/SchematicStatsResponse" }
              }
            }
          },
          "401": { "$ref": "#/components/responses/Unauthorized" },
          "403": {
            "description": "Not the schematic owner",
            "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Error" } } }
          },
          "404": { "$ref": "#/components/responses/NotFound" },
          "429": { "$ref": "#/components/responses/RateLimited" }
        }
      }
    },
    "/api/user/stats": {
      "get": {
        "operationId": "getUserStats",
        "summary": "Get user analytics",
        "description": "Returns aggregate hourly analytics across all schematics owned by the API key holder, plus a paginated schematic list with per-schematic totals. API key authentication only (HMAC not supported).",
        "tags": ["Analytics"],
        "security": [
          { "ApiKeyHeader": [] },
          { "ApiKeyQuery": [] }
        ],
        "parameters": [
          { "name": "page", "in": "query", "schema": { "type": "integer", "default": 1, "minimum": 1 }, "description": "Page number for the schematic list" }
        ],
        "responses": {
          "200": {
            "description": "User analytics data",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/UserStatsResponse" }
              }
            }
          },
          "401": { "$ref": "#/components/responses/Unauthorized" },
          "429": { "$ref": "#/components/responses/RateLimited" }
        }
      }
    },
    "/api/mod/download": {
      "post": {
        "operationId": "modDownload",
        "summary": "Download schematic for in-game mod",
        "description": "Downloads an XOR-encoded NBT file for the Create mod Minecraft integration. Requires HMAC authentication only.",
        "tags": ["Mod Integration"],
        "security": [
          { "HMACSignature": [] }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["message", "signature", "type"],
                "properties": {
                  "message": { "type": "string", "description": "Colon-separated: timestamp:modversion:mcusername:identifier" },
                  "signature": { "type": "string", "description": "HMAC-SHA256 hex signature of the message" },
                  "type": { "type": "string", "enum": ["name", "id"], "description": "Whether identifier is a schematic name or ID" }
                }
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "XOR-encoded NBT file bytes",
            "content": { "application/octet-stream": { "schema": { "type": "string", "format": "binary" } } }
          },
          "401": { "$ref": "#/components/responses/Unauthorized" },
          "404": { "$ref": "#/components/responses/NotFound" },
          "429": { "$ref": "#/components/responses/RateLimited" }
        }
      }
    }
  },
  "components": {
    "securitySchemes": {
      "ApiKeyHeader": {
        "type": "apiKey",
        "in": "header",
        "name": "X-API-Key",
        "description": "API key passed via X-API-Key header. Generate at /settings/api-keys."
      },
      "ApiKeyQuery": {
        "type": "apiKey",
        "in": "query",
        "name": "api_key",
        "description": "API key passed as a query parameter (less secure, use header when possible)."
      },
      "HMACSignature": {
        "type": "http",
        "scheme": "bearer",
        "description": "HMAC-SHA256 authentication via X-Mod-Message and X-Mod-Signature headers. Message format: timestamp:modversion:mcusername:identifier. The signature is HMAC-SHA256(message, shared_secret) hex-encoded. Timestamps must be within 5 minutes."
      }
    },
    "schemas": {
      "Error": {
        "type": "object",
        "properties": {
          "error": { "type": "string" }
        },
        "required": ["error"]
      },
      "HourlyStat": {
        "type": "object",
        "properties": {
          "hour": { "type": "string", "example": "2026-04-29 14", "description": "Hour bucket in YYYY-MM-DD HH format (UTC)" },
          "count": { "type": "integer", "format": "int64", "description": "Value for this hour" }
        },
        "required": ["hour", "count"]
      },
      "Schematic": {
        "type": "object",
        "properties": {
          "id": { "type": "string" },
          "name": { "type": "string" },
          "title": { "type": "string" },
          "author": { "type": "string" },
          "content": { "type": "string" },
          "excerpt": { "type": "string" },
          "featuredImage": { "type": "string" },
          "gallery": { "type": "array", "items": { "type": "string" } },
          "video": { "type": "string" },
          "categories": { "type": "array", "items": { "type": "string" } },
          "tags": { "type": "array", "items": { "type": "string" } },
          "views": { "type": "integer" },
          "downloads": { "type": "integer" },
          "rating": { "type": "number", "format": "float" },
          "blockCount": { "type": "integer" },
          "dimensions": {
            "type": "object",
            "properties": {
              "x": { "type": "integer" },
              "y": { "type": "integer" },
              "z": { "type": "integer" }
            }
          },
          "materials": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "block_id": { "type": "string" },
                "name": { "type": "string" },
                "count": { "type": "integer" }
              }
            }
          },
          "mods": { "type": "array", "items": { "type": "string" } },
          "created": { "type": "string", "format": "date-time" }
        }
      },
      "SchematicListResponse": {
        "type": "object",
        "properties": {
          "items": { "type": "array", "items": { "$ref": "#/components/schemas/Schematic" } },
          "page": { "type": "integer" },
          "pageSize": { "type": "integer" },
          "hasPrev": { "type": "boolean" },
          "hasNext": { "type": "boolean" },
          "total": { "type": "integer" },
          "totalPages": { "type": "integer" },
          "term": { "type": "string" }
        },
        "required": ["items", "page", "pageSize", "hasPrev", "hasNext", "total", "totalPages"]
      },
      "HomeResponse": {
        "type": "object",
        "properties": {
          "trending": { "type": "array", "items": { "$ref": "#/components/schemas/Schematic" } },
          "latest": { "type": "array", "items": { "$ref": "#/components/schemas/Schematic" } },
          "highestRated": { "type": "array", "items": { "$ref": "#/components/schemas/Schematic" } }
        },
        "required": ["trending", "latest", "highestRated"]
      },
      "FiltersResponse": {
        "type": "object",
        "properties": {
          "categories": { "type": "array", "items": { "type": "object", "properties": { "key": { "type": "string" }, "name": { "type": "string" }, "count": { "type": "integer" } } } },
          "minecraftVersions": { "type": "array", "items": { "type": "string" } },
          "createVersions": { "type": "array", "items": { "type": "object", "properties": { "group": { "type": "string" }, "value": { "type": "string" }, "versions": { "type": "array", "items": { "type": "string" } } } } },
          "tags": { "type": "array", "items": { "type": "object", "properties": { "key": { "type": "string" }, "name": { "type": "string" }, "count": { "type": "integer" } } } },
          "mods": { "type": "array", "items": { "type": "object", "properties": { "namespace": { "type": "string" }, "name": { "type": "string" }, "count": { "type": "integer" } } } }
        }
      },
      "Comment": {
        "type": "object",
        "properties": {
          "ID": { "type": "string" },
          "Published": { "type": "string" },
          "AuthorUsername": { "type": "string" },
          "AuthorAvatar": { "type": "string" },
          "Content": { "type": "string" },
          "Indent": { "type": "integer" },
          "ParentID": { "type": "string" },
          "ReplyToAuthor": { "type": "string" }
        }
      },
      "CommentsResponse": {
        "type": "object",
        "properties": {
          "count": { "type": "integer" },
          "comments": { "type": "array", "items": { "$ref": "#/components/schemas/Comment" } }
        },
        "required": ["count", "comments"]
      },
      "ChangesResponse": {
        "type": "object",
        "properties": {
          "changes": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "name": { "type": "string" },
                "kind": { "type": "string", "enum": ["updated", "removed"] }
              }
            }
          },
          "cursor": { "type": "string", "description": "Opaque token; pass this back as ?cursor= on the next call" },
          "hasMore": { "type": "boolean" }
        },
        "required": ["changes", "cursor", "hasMore"]
      },
      "StatItem": {
        "type": "object",
        "properties": {
          "name": { "type": "string" },
          "views": { "type": "integer" },
          "downloads": { "type": "integer" },
          "rating": { "type": "number", "format": "float" },
          "ratingCount": { "type": "integer" },
          "commentCount": { "type": "integer" }
        },
        "required": ["name", "views", "downloads", "rating", "ratingCount", "commentCount"]
      },
      "UploadResponse": {
        "type": "object",
        "properties": {
          "token": { "type": "string" },
          "url": { "type": "string" },
          "checksum": { "type": "string" },
          "filename": { "type": "string" },
          "size": { "type": "integer" },
          "dimensions": {
            "type": "object",
            "properties": {
              "x": { "type": "integer" },
              "y": { "type": "integer" },
              "z": { "type": "integer" }
            }
          },
          "block_count": { "type": "integer" },
          "materials": { "type": "array", "items": { "type": "object" } },
          "mods": { "type": "array", "items": { "type": "string" } }
        }
      },
      "SchematicStatsResponse": {
        "type": "object",
        "properties": {
          "name": { "type": "string" },
          "title": { "type": "string" },
          "total_views": { "type": "integer" },
          "total_downloads": { "type": "integer" },
          "comments": { "type": "integer", "format": "int64" },
          "vd_ratio": { "type": "number", "format": "float", "description": "Downloads / views ratio (0 to 1)" },
          "site_avg_vd_ratio": { "type": "number", "format": "float", "description": "Site-wide average downloads/views ratio" },
          "has_video": { "type": "boolean" },
          "views": { "type": "array", "items": { "$ref": "#/components/schemas/HourlyStat" }, "description": "Hourly view counts for the last 30 days" },
          "downloads": { "type": "array", "items": { "$ref": "#/components/schemas/HourlyStat" }, "description": "Hourly download counts" },
          "video_plays": { "type": "array", "items": { "$ref": "#/components/schemas/HourlyStat" }, "description": "Hourly video play counts (only present if has_video is true)" },
          "yt_clicks": { "type": "array", "items": { "$ref": "#/components/schemas/HourlyStat" }, "description": "Hourly YouTube click counts (only present if has_video is true)" },
          "time_on_page": { "type": "array", "items": { "$ref": "#/components/schemas/HourlyStat" }, "description": "Average time on page per hour (seconds)" },
          "layer_views": { "type": "array", "items": { "$ref": "#/components/schemas/HourlyStat" }, "description": "Hourly layer viewer usage counts" }
        },
        "required": ["name", "title", "total_views", "total_downloads", "comments", "vd_ratio", "site_avg_vd_ratio", "has_video", "views", "downloads", "time_on_page", "layer_views"]
      },
      "UserStatsSchematic": {
        "type": "object",
        "properties": {
          "name": { "type": "string" },
          "title": { "type": "string" },
          "featured_image": { "type": "string" },
          "views": { "type": "integer" },
          "downloads": { "type": "integer" },
          "vd_ratio": { "type": "number", "format": "float" },
          "created": { "type": "string", "format": "date-time" }
        }
      },
      "UserStatsResponse": {
        "type": "object",
        "properties": {
          "total_views": { "type": "integer" },
          "total_downloads": { "type": "integer" },
          "vd_ratio": { "type": "number", "format": "float" },
          "site_avg_vd_ratio": { "type": "number", "format": "float" },
          "views": { "type": "array", "items": { "$ref": "#/components/schemas/HourlyStat" } },
          "downloads": { "type": "array", "items": { "$ref": "#/components/schemas/HourlyStat" } },
          "video_plays": { "type": "array", "items": { "$ref": "#/components/schemas/HourlyStat" } },
          "yt_clicks": { "type": "array", "items": { "$ref": "#/components/schemas/HourlyStat" } },
          "time_on_page": { "type": "array", "items": { "$ref": "#/components/schemas/HourlyStat" } },
          "layer_views": { "type": "array", "items": { "$ref": "#/components/schemas/HourlyStat" } },
          "schematics": { "type": "array", "items": { "$ref": "#/components/schemas/UserStatsSchematic" } },
          "total_schematics": { "type": "integer" },
          "page": { "type": "integer" },
          "page_size": { "type": "integer" },
          "has_next": { "type": "boolean" },
          "has_prev": { "type": "boolean" }
        },
        "required": ["total_views", "total_downloads", "vd_ratio", "site_avg_vd_ratio", "views", "downloads", "video_plays", "yt_clicks", "time_on_page", "layer_views", "schematics", "total_schematics", "page", "page_size", "has_next", "has_prev"]
      }
    },
    "responses": {
      "BadRequest": {
        "description": "Invalid parameters or missing required fields",
        "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Error" } } }
      },
      "Unauthorized": {
        "description": "Missing or invalid authentication",
        "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Error" } } }
      },
      "NotFound": {
        "description": "Resource not found",
        "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Error" } } }
      },
      "RateLimited": {
        "description": "Rate limit exceeded",
        "headers": {
          "Retry-After": { "schema": { "type": "integer" }, "description": "Seconds to wait before retrying" }
        },
        "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Error" } } }
      }
    }
  }
}`

func OpenAPIHandler() func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		e.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
		e.Response.Header().Set("Access-Control-Allow-Origin", "*")
		e.Response.WriteHeader(http.StatusOK)
		_, err := e.Response.Write([]byte(openAPISpec))
		return err
	}
}
