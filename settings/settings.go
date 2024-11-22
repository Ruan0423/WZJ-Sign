package settings
import (
	"github.com/spf13/viper"
	"fmt"
)

var Conf = new(APPconfig)
type APPconfig struct {
	*APP `mapstructure:"app"`
	*Position `mapstructure:"position"`
}

type APP struct {
	Name string `mapstructure:"name"`
	Model string `mapstructure:"model"`
	Version string `mapstructure:"version"`
	Port int `mapstructure:"port"`
}

type Position struct {
	Lat string `mapstructure:"lat"`
	Lon string `mapstructure:"lon"`
}

func Init() (err error){
	// 使用viper加载配置。
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	err = viper.ReadInConfig()
	if err!= nil {
        fmt.Println("config init erro:",err)
		return err
    }

	if err = viper.Unmarshal(Conf); err != nil {

		fmt.Println("config init erro:",err)
		return err
	}
	fmt.Println("test config",Conf)
	return nil

}