package models

import (
	"time"

	"github.com/goravel/framework/database/orm"
	"golang.org/x/crypto/bcrypt"
)

// User contiene la identidad, autenticación y preferencias del usuario.
// El perfil profesional se guarda en Publicador o Chofer y se vincula mediante
// el identificador correspondiente.
type User struct {
	orm.Model

	// ── Identidad y acceso ──
	Name     string `gorm:"column:name;type:varchar(100);not null"`
	Email    string `gorm:"column:email;type:varchar(100);uniqueIndex;not null"`
	Password string `gorm:"column:password;type:varchar(255);not null"`
	Role     string `gorm:"column:role;type:varchar(50);index"` // admin | publicador | chofer

	// ── Contacto ──
	Phone          *string `gorm:"column:phone;type:varchar(30)"`
	PhoneAlt       *string `gorm:"column:phone_alt;type:varchar(30)"`
	WhatsApp       *string `gorm:"column:whatsapp;type:varchar(30)"`
	Telegram       *string `gorm:"column:telegram;type:varchar(60)"`
	EmergencyName  *string `gorm:"column:emergency_name;type:varchar(100)"`
	EmergencyPhone *string `gorm:"column:emergency_phone;type:varchar(30)"`

	// ── Empresa a la que pertenece el usuario (opcional) ──
	EmpresaID *uint    `gorm:"column:empresa_id;index"`
	Empresa   *Empresa `gorm:"foreignKey:EmpresaID"`

	// ── Vínculo con el perfil de rol (exactamente uno debe estar seteado) ──
	PublicadorID *uint       `gorm:"column:publicador_id;index"`
	Publicador   *Publicador `gorm:"foreignKey:PublicadorID"`

	ChoferID *uint   `gorm:"column:chofer_id;index"`
	Chofer   *Chofer `gorm:"foreignKey:ChoferID"`

	// ── Administración ──
	IsActive  bool       `gorm:"column:is_active;default:true;index"`
	LastLogin *time.Time `gorm:"column:last_login"`
	CreatedBy *uint      `gorm:"column:created_by"`
	UpdatedBy *uint      `gorm:"column:updated_by"`

	// ── Ubicación personal ──
	Address    string  `gorm:"column:address;type:text"`
	City       string  `gorm:"column:city;type:varchar(100)"`
	State      string  `gorm:"column:state;type:varchar(100)"`
	Country    string  `gorm:"column:country;type:varchar(100);default:Cuba"`
	PostalCode string  `gorm:"column:postal_code;type:varchar(20)"`
	Latitude   float64 `gorm:"column:latitude;type:decimal(10,7)"`
	Longitude  float64 `gorm:"column:longitude;type:decimal(10,7)"`
	Radius     int     `gorm:"column:radius;type:int;default:100"`

	// ── Preferencias de carga ──
	PreferredEquipmentTypes string  `gorm:"column:preferred_equipment_types;type:text"`
	PreferredCargoTypes     string  `gorm:"column:preferred_cargo_types;type:text"`
	MaxWeight               float64 `gorm:"column:max_weight;type:decimal(10,2)"`
	MaxDistance             float64 `gorm:"column:max_distance;type:decimal(10,2)"`
	PreferredRoutes         string  `gorm:"column:preferred_routes;type:text"`

	// ── Disponibilidad ──
	AvailableFrom *time.Time `gorm:"column:available_from"`
	AvailableTo   *time.Time `gorm:"column:available_to"`
	Notes         string     `gorm:"column:notes;type:text"`
}

// TableName fuerza el nombre público de la tabla para evitar ambigüedades con
// auth.users cuando la aplicación usa Supabase como PostgreSQL administrado.
func (User) TableName() string { return "users" }

// SetPassword reemplaza la contraseña por un hash bcrypt. Nunca almacena texto
// plano ni debe recibir un valor que ya esté hasheado.
func (u *User) SetPassword(plain string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hash)
	return nil
}

// CheckPassword compara una contraseña recibida con el hash almacenado.
func (u *User) CheckPassword(plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(plain)) == nil
}

func (u *User) IsAdmin() bool      { return u.Role == "admin" }
func (u *User) IsPublicador() bool { return u.Role == "publicador" }
func (u *User) IsChofer() bool     { return u.Role == "chofer" }
