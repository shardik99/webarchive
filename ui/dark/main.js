let globalData = [];
let viewMode = localStorage.getItem('viewMode') || 'grid';
let currentSearch = '';
let currentTag = '';

function initApp() {
    $("#site_title_sidebar").html("WebArchive " + window.location.hostname);
    document.title = "WebArchive " + window.location.hostname;
    
    // Setup View Toggles
    $('#btn_grid_view').on('click', () => setViewMode('grid'));
    $('#btn_list_view').on('click', () => setViewMode('list'));
    $('#btn_add_archive').on('click', showNewArchiveModal);
    
    // Setup Search
    $('#search_input').on('input', function() {
        currentSearch = $(this).val().toLowerCase();
        renderArchives();
    });

    if (window.location.pathname.endsWith("/")) {
        loadIndex();
    } else {
        loadPage(window.location.pathname.slice(1));
    }
}

function setViewMode(mode) {
    viewMode = mode;
    localStorage.setItem('viewMode', mode);
    
    $('.toggle-btn').removeClass('active');
    $(`#btn_${mode}_view`).addClass('active');
    
    // Only re-render if we are on the index page
    if (window.location.pathname.endsWith("/")) {
        renderArchives();
    }
}

function filterByTag(tag) {
    currentTag = tag;
    history.pushState(null, null, "/");
    renderArchives();
}

function goHome() {
    history.pushState(null, null, "/");
    $('#search_input').val('');
    currentSearch = '';
    currentTag = '';
    loadIndex();
}

function loadIndex() {
    // Show topbar elements if hidden
    $('#search_input').parent().show();
    $('#view_toggles_container').show();
    
    $.ajax({
        url: "/api/v1/pages", 
        success: function (data, status) {
            if (status !== "success") {
                gotError(status);
                return;
            }
            globalData = data;
            
            // Extract unique tags and render sidebar
            let allTags = new Set();
            if (globalData) {
                globalData.forEach(v => {
                    if (v.tags) v.tags.forEach(t => allTags.add(t));
                });
            }
            let tagsHtml = '';
            Array.from(allTags).sort().forEach(tag => {
                let activeClass = (tag === currentTag) ? 'active' : '';
                tagsHtml += `<li class="nav-item ${activeClass}" onclick="filterByTag('${tag}')"># ${tag}</li>`;
            });
            $('#tags_container').html(tagsHtml);

            // Ensure toggle buttons reflect current state
            $('.toggle-btn').removeClass('active');
            $(`#btn_${viewMode}_view`).addClass('active');
            renderArchives();
        }
    });
}

function renderArchives() {
    let elem = document.getElementById("data");
    elem.innerHTML = "";
    
    // Remove both layout classes before adding the correct one
    $(elem).removeClass("view-grid view-list");
    $(elem).addClass(viewMode === 'grid' ? "view-grid" : "view-list");

    const tmplId = viewMode === 'grid' ? 'card_tmpl' : 'list_row_tmpl';
    const tmpl = document.getElementById(tmplId);

    // Update active state in sidebar tags
    $('#tags_container .nav-item').removeClass('active');
    if (currentTag) {
        $(`#tags_container .nav-item:contains('# ${currentTag}')`).addClass('active');
    }

    globalData.forEach(function (v) {
        // Tag filtering
        if (currentTag && (!v.tags || !v.tags.includes(currentTag))) {
            return;
        }

        // Client-side search filtering
        if (currentSearch) {
            const titleMatch = v.meta.title && v.meta.title.toLowerCase().includes(currentSearch);
            const urlMatch = v.url && v.url.toLowerCase().includes(currentSearch);
            if (!titleMatch && !urlMatch) return;
        }

        let item_elem = tmpl.content.cloneNode(true);
        let container = $(item_elem).find('.item-container');
        
        container.attr("onclick", "goToPage('" + v.id + "');");
        
        // Status Badge
        let statusElem = $(item_elem).find(".status-badge");
        statusElem.addClass(v.status.toLowerCase());
        statusElem.html(v.status);
        statusElem.attr("title", v.status);
        
        // Text Fields
        $(item_elem).find(".created-text").html(v.created ? new Date(v.created).toLocaleDateString() : 'Unknown date');
        $(item_elem).find(".title-text").html(v.meta.title || 'Untitled');
        $(item_elem).find(".url-text").html(v.url);
        
        // Description and Tags (only in card view usually)
        if (viewMode === 'grid') {
            $(item_elem).find(".desc-text").html(v.meta.description || '');
            
            // Render Tags
            let tagsContainer = $(item_elem).find(".tags-container");
            if (v.tags && v.tags.length > 0) {
                let tHtml = '';
                v.tags.forEach(t => {
                    tHtml += `<span style="background: rgba(255,255,255,0.1); padding: 2px 8px; border-radius: 4px; font-size: 11px;">${t}</span>`;
                });
                tagsContainer.html(tHtml);
            }
        }

        elem.append(item_elem);
    });
}

function goToPage(id) {
    history.pushState({"page": id}, null, "/" + id);
    loadPage(id);
}

function loadPage(id) {
    // Hide topbar elements when viewing a single page
    $('#search_input').parent().hide();
    $('#view_toggles_container').hide();

    $.ajax({
        url: "/api/v1/pages/" + id, 
        success: function (data, status) {
            if (status !== "success") {
                gotError(status);
                return;
            }

            let elem = document.getElementById("data");
            elem.innerHTML = "";
            $(elem).removeClass("view-grid view-list"); // Clear layout classes

            let page_tmpl = document.getElementById("page_tmpl");
            let page_elem = page_tmpl.content.cloneNode(true);
            
            $(page_elem).find("#page_title").html(data.meta.title || 'Untitled');
            $(page_elem).find("#page_description").html(data.meta.description || '');
            $(page_elem).find("#page_url").html(data.url);

            const resultsContainer = $(page_elem).find("#results");
            const result_tmpl = document.getElementById("result_tmpl");

            if (data.results) {
                data.results.forEach(function (result) {
                    let result_elem = result_tmpl.content.cloneNode(true);
                    let badge = $(result_elem).find(".format-badge");
                    let link = $(result_elem).find(".result-link");

                    badge.html(result.format);
                    
                    if (result.error && result.error !== "") {
                        badge.addClass("error");
                        link.html("⚠ Error Occurred");
                        link.attr("title", result.error);
                    } else if (result.files && result.files.length > 0) {
                        result.files.forEach(function (file) {
                            // If multiple files, we could clone the card, but usually there's one main file per result
                            link.attr("onclick", "window.open('/api/v1/pages/" + data.id + "/file/" + file.id + "', '_blank');");
                            link.html(file.name);
                        });
                    }

                    resultsContainer.append(result_elem);
                });
            }

            elem.append(page_elem);
        }
    });
}

function gotError(err) {
    console.error("API Error:", err);
    $('#data').html('<div style="padding: 24px; color: var(--error);">Failed to load data. See console for details.</div>');
}

document.addEventListener("DOMContentLoaded", initApp);

window.addEventListener('popstate', function (event) {
    if (event.state === null || !event.state.page) {
        loadIndex();
    } else {
        loadPage(event.state.page);
    }
});



// New Archive Logic
function showNewArchiveModal() {
    $('#new_archive_modal').css('display', 'flex');
    $('#new_archive_url').val('');
    $('#new_archive_tags').val('');
    $('#new_archive_url').focus();
}

function hideNewArchiveModal() {
    $('#new_archive_modal').css('display', 'none');
}

function submitNewArchive() {
    const url = $('#new_archive_url').val().trim();
    const tagsStr = $('#new_archive_tags').val().trim();
    if (!url) return;

    let tags = [];
    if (tagsStr) {
        tags = tagsStr.split(',').map(t => t.trim()).filter(t => t);
    }

    let payload = {
        url: url,
        tags: tags
    };

    $.ajax({
        url: "/api/v1/pages",
        type: "POST",
        contentType: "application/json",
        data: JSON.stringify(payload),
        success: function() {
            hideNewArchiveModal();
            loadIndex(); // Refresh the list
        },
        error: function(err) {
            console.error("Failed to add archive:", err);
            alert("Failed to add archive. Check console.");
        }
    });
}

