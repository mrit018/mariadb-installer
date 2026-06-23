// Package examples รวมตัวอย่างการสั่งงาน mariadb-installer ในสถานการณ์ต่าง ๆ
// แสดงผ่าน flag --examples (เช่น `mariadb-installer.exe --examples`) เพื่อให้ผู้ใช้
// เห็นตัวอย่างคำสั่งจริงโดยไม่ต้องเปิด README แยก
//
// เพิ่มตัวอย่างใหม่ได้ง่าย ๆ ด้วยการเติม Example เข้าไปใน All() — ทุกตัวอย่างต้อง
// ระบุ Category เพื่อให้ List/Filter จัดกลุ่มแสดงผลได้
package examples

import (
	"fmt"
	"sort"
	"strings"
)

// Category คือหมวดหมู่ของตัวอย่าง ใช้จัดกลุ่มตอนแสดงผลและตอน filter ด้วย --examples=<category>
type Category string

const (
	CategorySingleHost   Category = "single-host"
	CategoryGalera       Category = "galera"
	CategoryTroubleshoot Category = "troubleshoot"
)

// categoryOrder กำหนดลำดับการแสดงผลของแต่ละหมวด (จากพื้นฐานไปซับซ้อน)
var categoryOrder = []Category{CategorySingleHost, CategoryGalera, CategoryTroubleshoot}

var categoryTitle = map[Category]string{
	CategorySingleHost:   "เครื่องเดียว (Single Host)",
	CategoryGalera:       "Galera Cluster (หลายเครื่อง)",
	CategoryTroubleshoot: "Troubleshooting / แก้ปัญหาเฉพาะกรณี",
}

// Example คือตัวอย่างการสั่งงานหนึ่งตัวอย่าง
type Example struct {
	Category    Category
	Title       string   // หัวข้อสั้น ๆ บอกว่าตัวอย่างนี้ทำอะไร
	Description string   // อธิบายเพิ่มเติม 1-2 บรรทัด (เมื่อใช้ตัวอย่างนี้ / ข้อควรระวัง)
	Commands    []string // คำสั่งจริงที่รัน (อาจมีมากกว่า 1 บรรทัด เช่นสร้างไฟล์ config ก่อนแล้วค่อยรัน)
}

// All คืนตัวอย่างทั้งหมดตามลำดับที่กำหนดไว้ (ไม่ได้ sort ตามตัวอักษร เพราะต้องการ
// เรียงจากเคสพื้นฐานไปเคสซับซ้อน/ผิดปกติ)
func All() []Example {
	var all []Example
	all = append(all, singleHostExamples()...)
	all = append(all, galeraExamples()...)
	all = append(all, troubleshootExamples()...)
	return all
}

// Filter คืนตัวอย่างที่ตรงกับ category ที่ระบุ (รับ string เพื่อให้ map ตรงกับ flag value ได้ง่าย)
// ถ้า category ว่างเปล่า คืนตัวอย่างทั้งหมด ถ้าระบุ category ที่ไม่มีจริง คืน slice ว่างพร้อม false
func Filter(categoryRaw string) ([]Example, bool) {
	if categoryRaw == "" {
		return All(), true
	}
	want := Category(strings.ToLower(strings.TrimSpace(categoryRaw)))
	if _, ok := categoryTitle[want]; !ok {
		return nil, false
	}
	var out []Example
	for _, ex := range All() {
		if ex.Category == want {
			out = append(out, ex)
		}
	}
	return out, true
}

// AvailableCategories คืนชื่อ category ทั้งหมดที่ใช้กับ --examples=<category> ได้ (เรียงตาม categoryOrder)
func AvailableCategories() []string {
	out := make([]string, len(categoryOrder))
	for i, c := range categoryOrder {
		out[i] = string(c)
	}
	return out
}

// Print พิมพ์ตัวอย่างทั้งหมดที่ส่งมา จัดกลุ่มตาม Category พร้อมหัวข้อคั่นให้อ่านง่าย
//
// แต่ละ entry ใน Commands ถูกพิมพ์ตามลักษณะของมัน:
//   - บรรทัดที่ขึ้นต้นด้วย "#" คือ comment อธิบายขั้นตอน พิมพ์ตรง ๆ ไม่มี "$" นำหน้า
//   - entry ที่มีเครื่องหมาย newline อยู่ภายใน (เนื้อหาไฟล์ เช่น JSON หลายบรรทัด) พิมพ์แบบ
//     ครอบด้วยเส้นคั่น ไม่ใส่ "$" นำหน้าแต่ละบรรทัด เพื่อไม่ให้สับสนว่าเป็นคำสั่ง shell
//   - entry อื่น ๆ ถือเป็นคำสั่งจริงที่รันได้ พิมพ์นำด้วย "$ "
func Print(exList []Example) {
	grouped := map[Category][]Example{}
	for _, ex := range exList {
		grouped[ex.Category] = append(grouped[ex.Category], ex)
	}

	first := true
	for _, cat := range categoryOrder {
		items := grouped[cat]
		if len(items) == 0 {
			continue
		}
		if !first {
			fmt.Println()
		}
		first = false

		title := categoryTitle[cat]
		fmt.Printf("=== %s ===\n", title)
		for i, ex := range items {
			fmt.Printf("\n%d. %s\n", i+1, ex.Title)
			if ex.Description != "" {
				fmt.Printf("   %s\n", ex.Description)
			}
			printCommands(ex.Commands)
		}
	}
}

// printCommands พิมพ์ entry ของ Commands แต่ละตัวด้วย format ที่เหมาะกับเนื้อหา
// (comment / คำสั่งจริง / เนื้อหาไฟล์หลายบรรทัด) ตามที่อธิบายไว้ใน Print
func printCommands(cmds []string) {
	for _, cmd := range cmds {
		switch {
		case cmd == "":
			fmt.Println()
		case strings.HasPrefix(cmd, "#"):
			fmt.Printf("   %s\n", cmd)
		case strings.Contains(cmd, "\n"):
			fmt.Println("   ----------------------------------------")
			for _, line := range strings.Split(cmd, "\n") {
				fmt.Printf("   %s\n", line)
			}
			fmt.Println("   ----------------------------------------")
		default:
			fmt.Printf("   $ %s\n", cmd)
		}
	}
}

// sortedCategoryNames ใช้ในข้อความ error ตอน --examples=<category> พิมพ์ผิด เพื่อบอกชื่อที่ถูกต้องทั้งหมด
func sortedCategoryNames() []string {
	names := AvailableCategories()
	sort.Strings(names)
	return names
}

// ErrUnknownCategoryHint คืนข้อความช่วยเหลือเมื่อผู้ใช้ระบุ --examples=<category> ที่ไม่มีจริง
func ErrUnknownCategoryHint(input string) string {
	return fmt.Sprintf(
		"ไม่รู้จักหมวดหมู่ %q ใช้ได้แค่: %s (หรือไม่ระบุ category เพื่อดูทั้งหมด)",
		input, strings.Join(sortedCategoryNames(), ", "),
	)
}
