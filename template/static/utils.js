var blockNameMap = window.blockNameMap || {};

function fetchBlockNames(blockIds, callback) {
    if (!blockIds || blockIds.length === 0) {
        if (callback) callback();
        return;
    }
    try {
        var xhr = new XMLHttpRequest();
        xhr.open('POST', 'https://blocksitems.com/api/v1/blocks/lookup', true);
        xhr.setRequestHeader('Content-Type', 'application/json');
        xhr.onload = function() {
            if (xhr.status === 200) {
                try {
                    var result = JSON.parse(xhr.responseText);
                    if (result.data) {
                        blockNameMap = result.data;
                    }
                } catch(e) {}
            }
            if (callback) callback();
        };
        xhr.onerror = function() {
            if (callback) callback();
        };
        xhr.send(JSON.stringify({ block_ids: blockIds }));
    } catch(e) {
        if (callback) callback();
    }
}

function formatBlockName(blockId) {
    if (!blockId) return '';
    var parts = blockId.split(':');
    var name = parts.length > 1 ? parts[1] : parts[0];
    return name.replace(/_/g, ' ').replace(/\b\w/g, function(c) { return c.toUpperCase(); });
}

function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    var k = 1024;
    var sizes = ['B', 'KB', 'MB', 'GB'];
    var i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

function animateCopyButton(btn) {
    btn.classList.add('copy-success');
    setTimeout(function() {
        btn.classList.remove('copy-success');
    }, 2000);
}
