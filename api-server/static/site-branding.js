(() => {
    const brand = window.TM_BRAND || { name: "Trip Manager", version: 0 };
    const icon = document.querySelector('link[rel="icon"]');
    if (icon) icon.href = "/site-icon?v=" + brand.version;
    for (const header of document.querySelectorAll(".tm-public-brand")) {
        header.textContent = brand.name;
        if (brand.logo) {
            const image = document.createElement("img");
            image.src = "/site-logo?v=" + brand.version;
            image.alt = "";
            header.prepend(image);
        }
    }
    document.title = document.body.dataset.pageTitle
        ? document.body.dataset.pageTitle + " | " + brand.name
        : brand.name;
})();
