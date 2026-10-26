document.addEventListener("DOMContentLoaded", function() {
    const urlParams = new URLSearchParams(window.location.search);
    let activeTab = urlParams.get('tab');
    if (!activeTab) {
        activeTab = 'dashboard';
    }
    switchTab(activeTab);
});

function switchTab(tabId) {
    document.querySelectorAll('.tab-link').forEach(btn => btn.classList.remove('active'));
    document.querySelectorAll('.tab-content').forEach(content => content.classList.remove('active'));
    
    let targetContent = document.getElementById(tabId);
    if (targetContent) {
        targetContent.classList.add('active');
    }
    
    let targetBtn = document.querySelector(`.tab-link[onclick*="'${tabId}'"]`);
    if (targetBtn) {
        targetBtn.classList.add('active');
    }
    
    window.history.replaceState(null, '', '?tab=' + tabId);
}

function openBuyModal() { document.getElementById('buyModal').style.display = 'flex'; }
function closeBuyModal() { document.getElementById('buyModal').style.display = 'none'; }

function addItemRow() {
    const container = document.getElementById('itemsContainer');
    const row = document.createElement('div');
    row.className = 'item-row';
    row.innerHTML = `
        <input list="categoriesList" name="category" placeholder="Kategoria" required>
        <input type="text" name="name" placeholder="Nazwa" required>
        <input type="text" name="color" placeholder="Kolor">
        <input type="text" name="size" placeholder="Rozmiar">
        <input type="text" name="brand" placeholder="Marka">
        <button type="button" onclick="this.parentElement.remove()" style="background:#dc3545; color:white; border:none; cursor:pointer; border-radius:4px; padding: 0 15px;">X</button>
    `;
    container.appendChild(row);
}

function sellItem(id, name) {
    let price = prompt("Czy na pewno chcesz sprzedać produkt: " + name + "?\nJeśli tak, wpisz za ile został sprzedany (PLN):");
    if (price !== null && price.trim() !== "") {
        let numPrice = parseFloat(price.replace(',', '.'));
        if (!isNaN(numPrice)) {
            document.getElementById('sell_item_id').value = id;
            document.getElementById('sell_price').value = numPrice;
            document.getElementById('sellForm').submit();
        } else {
            alert("Błąd: Podano nieprawidłową kwotę.");
        }
    }
}

function setDateRange(period) {
    let today = new Date();
    let from = new Date();
    
    if (period === 'week') {
        let day = today.getDay() || 7; 
        from.setDate(today.getDate() - day + 1);
    } else if (period === 'month') {
        from = new Date(today.getFullYear(), today.getMonth(), 1);
    }
    
    const formatLocal = (d) => {
        let m = '' + (d.getMonth() + 1);
        let day = '' + d.getDate();
        if (m.length < 2) m = '0' + m;
        if (day.length < 2) day = '0' + day;
        return [d.getFullYear(), m, day].join('-');
    };
    
    document.getElementById('dateFrom').value = formatLocal(from);
    document.getElementById('dateTo').value = formatLocal(today);
    
    document.getElementById('statsForm').submit();
}