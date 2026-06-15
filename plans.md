### plan 1.1:
    ***in this codebase i want everything to use realtime updates for chat elements, sms, cards and any updates,
    but on busines chatpage, after client clicks ihave paid, the updates to occur in realtime in business chat page,
    order/booking card with button confirm payment, also when client leaves a review the review should appear 
    in business chat page wothout page refresh or reload or htmx reload/swapt use websocket for this also
    cards in realtime, remove old refrech page on each notification since we will be using ws for realtime updates: 
    by replacin the manual   setInterval(refreshPendingPayments, 3000); function calls with websocket in business chatpage;
    goal: replace old ajax fectch on chat page items [messages, bookings, and orders events] with realtime websocket for both client and business page

    ============ Done Implemented 


### plan 1.2: 
    bookings and orders page, htmx table updates fix, on htmx request pages are stacking on each other: on clicking next table page, crreating duplicates  looking like
    Orders Management

    Track and manage your customer orders
    [quick order button]

    Orders Management

    Track and manage your customer orders
    [quick order button]
    [cards]
    [filder row]
    [recent order table]           similar behavior in bookings page, make htmx updates as clean as payments dashboard page

    my idea:
        move the repeating parts into the static part of the page to avoid rerender on htmx targat swap

    this is the static part that repeats:
        Orders Management
        Track and manage your customer orders

        [quick order button]
    :> [!WARNING]
    > on reloading the page should also not have repeated elements

    :> [!NOTE]
    > Do the same for bookings page

    ============ Done Implemented

### plan 1.3 
        ***Done***
    ============ Done Implemented 

### plan 1.4
    the Typing animation and layout fix: the dots should be horizantal 
                                         no backgroun both for Typing text and the dots animations
    the Typing animation is not appearing in business chat page, fix the issue

    ============ Done Implemented

### plan 1.5
    in reports page, in revenue tab: add export revune info to csv beside the print button which only prints pdf
                    add implement functionality to export revuen by csv

### pland 1.6
    since we are usng websocket, remove the heart beat, use websocket to detect if client or business is online
    [GIN] 2026/06/14 - 22:25:57 | 200 |    8.024174ms |       127.0.0.1 | POST     "/client/heartbeat"
    [GIN] 2026/06/14 - 22:25:59 | 200 |    8.875199ms |       127.0.0.1 | POST     "/client/heartbeat"
    [GIN] 2026/06/14 - 22:26:04 | 200 |    8.313127ms |       127.0.0.1 | POST     "/client/heartbeat"
    [GIN] 2026/06/14 - 22:26:27 | 200 |    8.072004ms |       127.0.0.1 | POST     "/client/heartbeat"
    k

### plan 1.7
    use custom alert modal instead of native brower alert() on to confirm client delete in clientlist
    ensure averywhere uses a custom alert, predefined, not old native alert prompt, in both client and business chatpages
    even in business dashboard pages use custom alert modal

### plan 1.8
    in all the business subpages orders,bookings payments etc, where there is table, sort the table giving priority, 1, pending status, then date , with most resent items coming first, pening payments to come on top of completed recent payments, optimize the arrangement of deatails,
    Ensure not table has a max of table size env variable, some tables, in reports page have more items in one page, use multiple pages
    implement => table pages in reports tables for more data with more that table size vairiable like in other pages [payments, orders and bookings]

### plan 1.9
    on creating an order in business chat page, it creates an empty message,
    fix: make creating order like creating bookings in business chat page;
    goal: no extra empty messages, when crteating orders in business page, only the order card is created 

    also: add a cancel feature in order card: ins step1,
    while waiting for client aproval add cancel button feature like in bookings card,
        on click it canels the order
    goal;  [Cancel button] [Waiting Aproval] withe apropriate icons, and otiized font sizes and spacing, use best wording (use optimized styling)
    inspiraton: <div class="flex gap-2">
        <button class="flex-1 py-1.5 sm:py-2 px-2 sm:px-3 rounded-lg border border-[var(--color-error)]/40 text-[var(--color-error)] hover:bg-[var(--color-error-light)] text-xs sm:text-sm font-medium transition" onclick="cancelOrder( 33 )">
          <i class="fas fa-times mr-1"></i>Cancel
        </button>
        
        <div class="flex-[2] py-1.5 sm:py-2 px-2 sm:px-3 rounded-lg bg-[var(--color-info-light)] text-[var(--color-info)] text-xs sm:text-sm font-medium text-center">
          <i class="fas fa-clock mr-1"></i>Waiting Approval
        </div>
        
      </div>
