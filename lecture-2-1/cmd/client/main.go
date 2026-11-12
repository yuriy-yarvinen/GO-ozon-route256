package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	deprecatedexample "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/examples/deprecated"
	enumexample "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/examples/enum"
	googlecommontypes "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/examples/googlecommontypes"
	mapsexample "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/examples/maps"
	oneofexample "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/examples/oneof"
	optionalexample "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/examples/optional"
	scalarvaluetypes "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/examples/scalarvaluetypes"
	examplesv1 "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/examples/v1"
	wellknowntypes "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/examples/wellknowntypes"
	greeterv1 "gitlab.ozon.dev/go/classroom-8/students/lecture-2-1/gen/greeter/v1"
	calendarperiod "google.golang.org/genproto/googleapis/type/calendarperiod"
	"google.golang.org/genproto/googleapis/type/color"
	"google.golang.org/genproto/googleapis/type/date"
	dayofweek "google.golang.org/genproto/googleapis/type/dayofweek"
	"google.golang.org/genproto/googleapis/type/decimal"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/genproto/googleapis/type/fraction"
	"google.golang.org/genproto/googleapis/type/latlng"
	"google.golang.org/genproto/googleapis/type/localized_text"
	"google.golang.org/genproto/googleapis/type/money"
	"google.golang.org/genproto/googleapis/type/month"
	phonenumber "google.golang.org/genproto/googleapis/type/phone_number"
	"google.golang.org/genproto/googleapis/type/postaladdress"
	"google.golang.org/genproto/googleapis/type/timeofday"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const defaultAddress = "localhost:50051"

func main() {
	name := "gRPC"
	if len(os.Args) > 1 {
		name = os.Args[1]
	}

	address := os.Getenv("GRPC_ADDR")
	if address == "" {
		address = defaultAddress
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		address,
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect to %s: %v", address, err)
	}
	defer conn.Close()

	greeting, err := greeterv1.NewGreeterClient(conn).SayHello(ctx, &greeterv1.SayHelloRequest{Name: name, Ggg: "gRPC"})
	if err != nil {
		log.Fatalf("failed to call SayHello: %v", err)
	}
	fmt.Println(greeting.GetMessage())

	client := examplesv1.NewExamplesClient(conn)
	scalarResponse, err := client.ScalarValueTypes(ctx, &scalarvaluetypes.ScalarValueTypesExample{
		DoubleValue: 1.25, FloatValue: 2.5, Int32Value: -32, Int64Value: -64,
		Uint32Value: 32, Uint64Value: 64, Sint32Value: -320, Sint64Value: -640,
		Fixed32Value: 320, Fixed64Value: 640, Sfixed32Value: -3200, Sfixed64Value: -6400,
		BoolValue: true, StringValue: "protobuf", BytesValue: []byte("bytes"),
	})
	printResult("ScalarValueTypes", scalarResponse, err)
	enumResponse, err := client.Enum(ctx, &enumexample.EnumExample{
		Status: enumexample.OrderStatus_ORDER_STATUS_PAID,
	})
	printResult("Enum", enumResponse, err)
	mapsResponse, err := client.Maps(ctx, &mapsexample.MapsExample{
		Labels: map[string]string{"environment": "demo", "language": "go"},
		LocationsById: map[int32]*mapsexample.Location{
			1: {Latitude: 55.7558, Longitude: 37.6173},
		},
	})
	printResult("Maps", mapsResponse, err)
	oneOfResponse, err := client.OneOf(ctx, &oneofexample.OneOfExample{
		Contact: &oneofexample.Contact{Value: &oneofexample.Contact_Email{Email: "user@example.com"}},
	})
	printResult("OneOf", oneOfResponse, err)
	optionalResponse, err := client.Optional(ctx, &optionalexample.OptionalExample{
		Nickname: proto.String("yuri"), Age: proto.Int32(30),
	})
	printResult("Optional", optionalResponse, err)
	deprecatedResponse, err := client.Deprecated(ctx, &deprecatedexample.DeprecatedExample{
		OldName: "legacy", Name: "current",
	})
	printResult("Deprecated", deprecatedResponse, err)
	deprecatedServiceResponse, err := deprecatedexample.NewDeprecatedServiceClient(conn).OldMethod(ctx, &deprecatedexample.DeprecatedExample{
		OldName: "legacy method", Name: "deprecated service",
	})
	printResult("DeprecatedService.OldMethod", deprecatedServiceResponse, err)

	payload, err := anypb.New(&structpb.Struct{Fields: map[string]*structpb.Value{
		"source": structpb.NewStringValue("client"),
	}})
	if err != nil {
		log.Fatalf("failed to create Any example payload: %v", err)
	}
	metadata, err := structpb.NewStruct(map[string]any{"source": "client", "enabled": true})
	if err != nil {
		log.Fatalf("failed to create Struct example metadata: %v", err)
	}
	wellKnownResponse, err := client.GoogleWellKnownTypes(ctx, &wellknowntypes.GoogleWellKnownTypesExample{
		CreatedAt: timestamppb.Now(),
		Timeout:   durationpb.New(30 * time.Second),
		Payload:   payload,
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"metadata", "optional_title"},
		},
		Metadata:      metadata,
		Value:         structpb.NewStringValue("example"),
		OptionalTitle: wrapperspb.String("Demo"),
	})
	printResult("GoogleWellKnownTypes", wellKnownResponse, err)
	if _, err := wellknowntypes.NewHealthCheckClient(conn).Check(ctx, &emptypb.Empty{}); err != nil {
		log.Fatalf("failed to call HealthCheck.Check: %v", err)
	}
	fmt.Println("HealthCheck.Check: OK")
	googleCommonResponse, err := client.GoogleCommonTypes(ctx, &googlecommontypes.GoogleCommonTypesExample{
		Date:  &date.Date{Year: 2026, Month: 11, Day: 13},
		Price: &money.Money{CurrencyCode: "RUB", Units: 100, Nanos: 500000000},
		Address: &postaladdress.PostalAddress{
			RegionCode: "RU", Locality: "Moscow", AddressLines: []string{"Red Square"},
		},
		Color:          &color.Color{Red: 0.2, Green: 0.4, Blue: 0.8, Alpha: wrapperspb.Float(1)},
		PhoneNumber:    &phonenumber.PhoneNumber{Kind: &phonenumber.PhoneNumber_E164Number{E164Number: "+79991234567"}},
		LocalizedText:  &localized_text.LocalizedText{Text: "Привет", LanguageCode: "ru"},
		Location:       &latlng.LatLng{Latitude: 55.7558, Longitude: 37.6173},
		DayOfWeek:      dayofweek.DayOfWeek_FRIDAY,
		Month:          month.Month_NOVEMBER,
		TimeOfDay:      &timeofday.TimeOfDay{Hours: 12, Minutes: 30},
		Fraction:       &fraction.Fraction{Numerator: 3, Denominator: 4},
		Decimal:        &decimal.Decimal{Value: "19.95"},
		Expression:     &expr.Expr{Expression: "price < 100", Title: "Under limit"},
		CalendarPeriod: calendarperiod.CalendarPeriod_MONTH,
	})
	printResult("GoogleCommonTypes", googleCommonResponse, err)
}

func printResult(name string, response proto.Message, err error) {
	if err != nil {
		log.Fatalf("failed to call %s: %v", name, err)
	}

	output, err := (protojson.MarshalOptions{Indent: "  "}).Marshal(response)
	if err != nil {
		log.Fatalf("failed to encode %s response: %v", name, err)
	}
	fmt.Printf("%s:\n%s\n", name, output)
}
