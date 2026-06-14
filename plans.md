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

### plan 1.3 
   in payments page, i removed analytics-stats-skeleton in payment page, its crreating an extra unwanted space, check if it break any code

### plan 1.4
    the Typing animation and layout fix: the dots should be horizantal 
                                         no backgroun both for Typing text and the dots animations
    the Typing animation is not appearing in business chat page, fix the issue

### plan 1.5
    in reports page, in revenue tab: add export revune info to csv beside the print button which only prints pdf
                    add implement functionality to export revuen by csv
