let globalData = [];
let globalCollections = [];
let viewMode = localStorage.getItem('viewMode') || 'grid';
let currentSearch = '';
let currentTag = '';
let currentTab = 'dashboard';
let activeCollectionId = null;

function initApp() {
    $("#site_title_sidebar").html("WebArchive " + window.location.hostname);
    document.title = "WebArchive " + window.location.hostname;
    
    // Setup View Toggles
    $('#btn_grid_view').on('click', () => setViewMode('grid'));
    $('#btn_list_view').on('click', () => setViewMode('list'));
    
    // Setup Search
    $('#search_input').on('input', function() {
        currentSearch = $(this).val().toLowerCase();
        renderArchives();
    });

    if (window.location.pathname.endsWith("/")) {
        loadData();
    } else {
        loadPage(window.location.pathname.slice(1));
    }
}

function setViewMode(mode) {
    viewMode = mode;
    localStorage.setItem('viewMode', mode);
    
    $('.toggle-btn').removeClass('active');
    $(`#btn_${mode}_view`).addClass('active');
    
    if (window.location.pathname.endsWith("/")) {
        renderArchives();
    }
}

function setTab(tab) {
    currentTab = tab;
    activeCollectionId = null;
    currentTag = '';
    
    $('.nav-item').removeClass('active');
    $(`#nav_${tab}`).addClass('active');
    
    if (window.location.pathname !== "/") {
        history.pushState(null, null, "/");
    }
    
    $('#search_input').parent().show();
    $('#view_toggles_container').show();
    
    renderArchives();
}

function viewCollection(id) {
    currentTab = 'collection_view';
    activeCollectionId = id;
    currentTag = '';
    
    $('.nav-item').removeClass('active');
    
    if (window.location.pathname !== "/") {
        history.pushState(null, null, "/");
    }
    
    $('#search_input').parent().show();
    $('#view_toggles_container').show();
    renderArchives();
}

function filterByTag(tag) {
    currentTag = tag;
    // When filtering by tag, usually we want to search across all links
    currentTab = 'all_links'; 
    $('.nav-item').removeClass('active');
    $(`#nav_all_links`).addClass('active');

    history.pushState(null, null, "/");
    renderArchives();
}

function goHome() {
    history.pushState(null, null, "/");
    $('#search_input').val('');
    currentSearch = '';
    currentTag = '';
    setTab('dashboard');
    loadData();
}

function loadData() {
    $('#search_input').parent().show();
    $('#view_toggles_container').show();
    
    Promise.all([
        $.ajax({ url: "/api/v1/pages" }),
        $.ajax({ url: "/api/v1/collections" })
    ]).then(([pages, collections]) => {
        globalData = pages || [];
        globalCollections = collections || [];
        
        let allTags = new Set();
        globalData.forEach(v => {
            if (v.tags) v.tags.forEach(t => allTags.add(t));
        });
        
        let tagsHtml = '';
        Array.from(allTags).sort().forEach(tag => {
            let activeClass = (tag === currentTag) ? 'active' : '';
            tagsHtml += `<li class="nav-item ${activeClass}" onclick="filterByTag('${tag}')"># ${tag}</li>`;
        });
        $('#tags_container').html(tagsHtml);

        $('.toggle-btn').removeClass('active');
        $(`#btn_${viewMode}_view`).addClass('active');
        
        if (['dashboard', 'all_links', 'collections'].includes(currentTab)) {
            $('.nav-item').removeClass('active');
            $(`#nav_${currentTab}`).addClass('active');
        }

        renderArchives();
    }).catch(err => {
        gotError(err);
    });
}

function renderArchives() {
    let elem = document.getElementById("data");
    elem.innerHTML = "";
    
    $(elem).removeClass("view-grid view-list");
    $(elem).addClass(viewMode === 'grid' ? "view-grid" : "view-list");

    const tmplId = viewMode === 'grid' ? 'card_tmpl' : 'list_row_tmpl';
    const tmpl = document.getElementById(tmplId);
    const colTmpl = document.getElementById("collection_card_tmpl");

    $('#tags_container .nav-item').removeClass('active');
    if (currentTag) {
        $(`#tags_container .nav-item:contains('# ${currentTag}')`).addClass('active');
    }

    let itemsToRender = [];
    
    if (currentTab === 'dashboard') {
        itemsToRender = [
            ...globalCollections.map(c => ({...c, isCollection: true})),
            ...globalData.filter(p => !p.collection_id)
        ];
    } else if (currentTab === 'all_links') {
        itemsToRender = [...globalData];
    } else if (currentTab === 'collections') {
        itemsToRender = globalCollections.map(c => ({...c, isCollection: true}));
    } else if (currentTab === 'collection_view') {
        itemsToRender = globalData.filter(p => p.collection_id === activeCollectionId);
    }
    
    itemsToRender.sort((a, b) => new Date(b.created) - new Date(a.created));

    itemsToRender.forEach(function (v) {
        if (v.isCollection) {
            if (currentSearch && !v.name.toLowerCase().includes(currentSearch)) return;
            
            let item_elem = colTmpl.content.cloneNode(true);
            let container = $(item_elem).find('.item-container');
            container.attr("onclick", "viewCollection('" + v.id + "');");
            
            $(item_elem).find(".created-text").html(v.created ? new Date(v.created).toLocaleDateString() : 'Unknown date');
            $(item_elem).find(".title-text").html(v.name || 'Untitled Collection');
            $(item_elem).find(".desc-text").html(v.description || '');
            elem.append(item_elem);
            return;
        }

        if (currentTag && (!v.tags || !v.tags.includes(currentTag))) {
            return;
        }

        if (currentSearch) {
            const titleMatch = v.meta && v.meta.title && v.meta.title.toLowerCase().includes(currentSearch);
            const urlMatch = v.url && v.url.toLowerCase().includes(currentSearch);
            if (!titleMatch && !urlMatch) return;
        }

        let item_elem = tmpl.content.cloneNode(true);
        let container = $(item_elem).find('.item-container');
        
        container.attr("onclick", "goToPage('" + v.id + "');");
        
        let statusElem = $(item_elem).find(".status-badge");
        statusElem.addClass(v.status.toLowerCase());
        statusElem.html(v.status);
        statusElem.attr("title", v.status);
        
        $(item_elem).find(".created-text").html(v.created ? new Date(v.created).toLocaleDateString() : 'Unknown date');
        $(item_elem).find(".title-text").html((v.meta && v.meta.title) || 'Untitled');
        $(item_elem).find(".url-text").html(v.url);
        
        if (viewMode === 'grid') {
            $(item_elem).find(".desc-text").html((v.meta && v.meta.description) || '');
            
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
            $(elem).removeClass("view-grid view-list");

            let page_tmpl = document.getElementById("page_tmpl");
            let page_elem = page_tmpl.content.cloneNode(true);
            
            $(page_elem).find("#page_title").html((data.meta && data.meta.title) || 'Untitled');
            $(page_elem).find("#page_description").html((data.meta && data.meta.description) || '');
            $(page_elem).find("#page_url").html(data.url);

            $(page_elem).find("#btn_edit_page").on('click', () => showEditArchiveModal(data));
            $(page_elem).find("#btn_delete_page").on('click', () => deleteArchive(data.id));

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
        loadData();
    } else {
        loadPage(event.state.page);
    }
});

// New Archive Logic
function showNewArchiveModal() {
    $('#new_archive_modal').css('display', 'flex');
    $('#new_archive_url').val('');
    $('#new_archive_tags').val('');
    $('#new_archive_depth').val('0');
    $('#new_archive_markdown').prop('checked', false);
    
    // Populate collections
    let colHtml = '<option value="" style="background: var(--bg-dark);">No Collection</option>';
    globalCollections.forEach(c => {
        colHtml += `<option value="${c.id}" style="background: var(--bg-dark);">${c.name}</option>`;
    });
    $('#new_archive_collection').html(colHtml);
    
    if (currentTab === 'collection_view' && activeCollectionId) {
        $('#new_archive_collection').val(activeCollectionId);
    }
    
    $('#new_archive_url').focus();
}

function hideNewArchiveModal() {
    $('#new_archive_modal').css('display', 'none');
}

function submitNewArchive() {
    const url = $('#new_archive_url').val().trim();
    const tagsStr = $('#new_archive_tags').val().trim();
    const collectionId = $('#new_archive_collection').val();
    const depthStr = $('#new_archive_depth').val();
    if (!url) return;

    let tags = [];
    if (tagsStr) {
        tags = tagsStr.split(',').map(t => t.trim()).filter(t => t);
    }

    let formats = ["single_file", "pdf"];
    if ($('#new_archive_markdown').is(':checked')) {
        formats.push("markdown");
    }

    let payload = {
        url: url,
        tags: tags,
        formats: formats,
        depth: parseInt(depthStr) || 0
    };
    
    if (collectionId) {
        payload.collection_id = collectionId;
    }

    $.ajax({
        url: "/api/v1/pages",
        method: "POST",
        data: JSON.stringify(payload),
        contentType: "application/json",
        success: function () {
            hideNewArchiveModal();
            loadData();
        },
        error: function(err) {
            console.error("Failed to add archive", err);
            alert("Failed to add archive.");
        }
    });
}

// New Collection Logic
function showNewCollectionModal() {
    $('#new_collection_modal').css('display', 'flex');
    $('#new_collection_name').val('');
    $('#new_collection_desc').val('');
    $('#new_collection_name').focus();
}

function hideNewCollectionModal() {
    $('#new_collection_modal').css('display', 'none');
}

function submitNewCollection() {
    const name = $('#new_collection_name').val().trim();
    const desc = $('#new_collection_desc').val().trim();
    if (!name) return;

    let payload = {
        name: name,
        description: desc
    };

    $.ajax({
        url: "/api/v1/collections",
        method: "POST",
        data: JSON.stringify(payload),
        contentType: "application/json",
        success: function () {
            hideNewCollectionModal();
            loadData();
        },
        error: function(err) {
            console.error("Failed to add collection", err);
            alert("Failed to add collection.");
        }
    });
}

// Edit Archive Logic
function showEditArchiveModal(data) {
    $('#edit_archive_id').val(data.id);
    $('#edit_archive_title').val(data.meta.title || '');
    $('#edit_archive_desc').val(data.meta.description || '');
    $('#edit_archive_tags').val((data.tags || []).join(', '));
    $('#edit_archive_modal').css('display', 'flex');
}

function hideEditArchiveModal() {
    $('#edit_archive_modal').css('display', 'none');
}

function submitEditArchive() {
    const id = $('#edit_archive_id').val();
    const title = $('#edit_archive_title').val().trim();
    const desc = $('#edit_archive_desc').val().trim();
    const tagsStr = $('#edit_archive_tags').val().trim();

    let tags = [];
    if (tagsStr) {
        tags = tagsStr.split(',').map(t => t.trim()).filter(t => t);
    }

    let payload = {
        title: title,
        description: desc,
        tags: tags
    };

    $.ajax({
        url: "/api/v1/pages/" + id,
        method: "PATCH",
        data: JSON.stringify(payload),
        contentType: "application/json",
        success: function () {
            hideEditArchiveModal();
            loadPage(id);
        },
        error: function(err) {
            console.error("Failed to edit archive", err);
            alert("Failed to edit archive.");
        }
    });
}

// Delete Archive Logic
function deleteArchive(id) {
    if (!confirm("Are you sure you want to permanently delete this archive and all its files?")) {
        return;
    }

    $.ajax({
        url: "/api/v1/pages/" + id,
        method: "DELETE",
        success: function () {
            history.pushState(null, null, "/");
            loadData();
        },
        error: function(err) {
            console.error("Failed to delete archive", err);
            alert("Failed to delete archive.");
        }
    });
}
