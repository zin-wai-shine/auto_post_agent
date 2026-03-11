package llm

import (
	"fmt"
)

// ListingData holds the data needed to generate marketing content.
type ListingData struct {
	Title       string
	Description string
	Property    string // condo, villa, etc
	ListingType string // sale, rent
	Price       float64
	Currency    string
	Bedrooms    int
	Bathrooms   int
	AreaSqm     float64
	Location    string
	Tags        []string
}

// GeneratedContent holds the LLM-generated trilingual text.
type GeneratedContent struct {
	Language string
	Title    string
	Body     string
}

// GenerateTrilingualContent takes listing specs and generates Facebook-optimized posts.
// In Phase 3 production, this connects to OpenAI/Ollama APIs. For the CLI demo,
// it uses an advanced templating engine to generate high-conversion output instantly.
func GenerateTrilingualContent(data ListingData) []GeneratedContent {
	priceStr := fmt.Sprintf("%.0f %s", data.Price, data.Currency)
	specs := ""
	if data.Bedrooms > 0 {
		specs += fmt.Sprintf("🛏️ %d Bed ", data.Bedrooms)
	}
	if data.Bathrooms > 0 {
		specs += fmt.Sprintf("🚿 %d Bath ", data.Bathrooms)
	}
	if data.AreaSqm > 0 {
		specs += fmt.Sprintf("📐 %.0f Sqm", data.AreaSqm)
	}

	// 1. English (Expat/Professional Focus)
	enTitle := "✨ Stunning " + data.Property + " for " + data.ListingType + " in " + data.Location
	enBody := fmt.Sprintf(`Don't miss out on this incredible %s in the heart of %s!
	
%s

%s
💰 Asking Price: %s

🌟 Key Features:
• Premium finishes and modern layout
• Unbeatable location with easy access to lifestyle amenities
• Perfect for professionals and families

Contact us today to schedule a private viewing. 📩

#%s #RealEstate #ThailandProperty #HouseHunting
`, data.Property, data.Location, data.Description, specs, priceStr, data.Property)

	// 2. Thai (Local/Casual Facebook Marketplace Focus)
	typeThai := "คอนโด"
	if data.Property == "villa" {
		typeThai = "พูลวิลล่า"
	}
	if data.Property == "house" {
		typeThai = "บ้านเดี่ยว"
	}
	if data.Property == "shophouse" {
		typeThai = "อาคารพาณิชย์"
	}

	actionThai := "ขาย"
	if data.ListingType == "rent" {
		actionThai = "ให้เช่า"
	}

	thTitle := "🔥 ด่วน! " + actionThai + typeThai + "สวย ทำเลดี " + data.Location
	thBody := fmt.Sprintf(`หลุดจอง! %sทำเลทอง %s พร้อมเข้าอยู่ทันที 🌟
	
รายละเอียด:
%s
%s
💵 ราคาเพียง: %s

✅ เดินทางสะดวก ใกล้สิ่งอำนวยความสะดวก
✅ เหมาะสำหรับอยู่อาศัย หรือลงทุนปล่อยเช่ากำไรดี
✅ นัดชมห้องจริงได้ทุกวัน ทักแชทเลย! 📩

#ขาย%s #อสังหาริมทรัพย์ #%s #ทำเลดี
`, typeThai, data.Location, data.Description, specs, priceStr, typeThai, data.Location)

	// 3. Myanmar (Investor Focus)
	myTitle := "🌟 အထူးအခွင့်အရေး! " + data.Location + " ရှိ " + data.Property + " လူနေမှုအဆင့်အတန်းမြင့်"
	myBody := fmt.Sprintf(`%s တွင် ရင်းနှီးမြှုပ်နှံရန် သို့မဟုတ် နေထိုင်ရန် အကောင်းဆုံး %s!

%s
%s
💰 ဈေးနှုန်း: %s

✨ အဓိကအချက်များ:
• ခေတ်မီပြီး သပ်ရပ်သော ဒီဇိုင်း
• သွားလာရလွယ်ကူသော နေရာကောင်း
• အမြတ်အစွန်းရရှိနိုင်သော အကောင်းဆုံးရင်းနှီးမြှုပ်နှံမှု

အသေးစိတ်သိရှိလိုပါက ယခုပဲ ဆက်သွယ်လိုက်ပါ။ 📩

#MyanmarInvestor #%s #PropertyInvestment
`, data.Location, data.Property, data.Description, specs, priceStr, data.Property)

	return []GeneratedContent{
		{Language: "en", Title: enTitle, Body: enBody},
		{Language: "th", Title: thTitle, Body: thBody},
		{Language: "my", Title: myTitle, Body: myBody},
	}
}
