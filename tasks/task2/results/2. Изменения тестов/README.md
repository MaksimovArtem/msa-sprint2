Данные для таблицы bookings перенесены в `init-fixtures2.sql`

regress.sh переработан так, чтобы проверять и получать данные БД1(база монолита) и БД2(база для bookings)
Данные по таблицам `app_user`, `hotel`, `review` и `promo_code` загружаются в БД1(не менялось), данные для `bookings` загружаются в БД2

Также из regress.sh удален тест на `GET /api/bookings`, тк монолит в GRPC режиме не может отправлять запросы без user_id
То есть запрос даже не может дойти до `booking-service`

Stacktrace:
```
2026-03-13T11:10:55.186Z ERROR 1 --- [io-8080-exec-12] o.a.c.c.C.[.[.[/].[dispatcherServlet]    : Servlet.service() for servlet [dispatcherServlet] in context with path [] threw exception [Request processing failed: java.lang.NullPointerException] with root cause

java.lang.NullPointerException: null
	at com.hotelio.proto.booking.BookingListRequest$Builder.setUserId(BookingListRequest.java:449) ~[p-o-y-1.0.0.jar!/:na]
	at com.hotelio.GrpcBookingService.listAll(GrpcBookingService.java:26) ~[p-o-y-1.0.0.jar!/:na]
	at com.hotelio.monolith.controller.BookingController.listBookings(BookingController.java:23) ~[!/:1.0.0]
        ...
```

