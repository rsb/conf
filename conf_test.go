package conf_test

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/rsb/conf"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type SomeFeatureConfig struct {
	DB
	LambdaHandler
	FeatureField string `conf:"env:SOME_FEATURE_FIELD,default:foo"`
	IgnoreField  int    `conf:"-"`
}

type DB struct {
	CLIHost string `conf:"env:CURRICULUM_DB_CLI_HOST,required"`
	Host    string `conf:"env:CURRICULUM_DB_HOST,required,cli:db-host,cli-s:h,cli-u:database host"`
	Name    string `conf:"env:CURRICULUM_DB_NAME,required"`
	User    string `conf:"env:CURRICULUM_DB_USER,required"`
	Port    int    `conf:"env:CURRICULUM_DB_PORT,default:5432"`
	Pass    string `conf:"env:CURRICULUM_DB_PASS,required"`
	IsDebug bool   `conf:"env:CURRICULUM_DB_DEBUG,default:true"`
	Logfile string `conf:"env:CURRICULUM_DB_LOGFILE,default:/curriculum.log"`
}

type LambdaHandler struct {
	AppName string `conf:"env:APP_NAME,required"`
}

type InvalidConfigTagParse struct {
	Value string `conf:"env:,default:,"`
}

type InvalidConfigEmptyEnv struct {
	Value string `conf:"default:foo"`
}

func TestProcessEnv_FieldsFailure(t *testing.T) {
	var config InvalidConfigTagParse

	err := conf.ProcessEnv(&config)
	require.Error(t, err, "conf.ProcessEnv is expected to fail")
	assert.Contains(t, err.Error(), "Fields failed: parseTag failed (Value)")
}

func TestProcessEnv_EmptyEnv(t *testing.T) {
	var config InvalidConfigEmptyEnv

	err := conf.ProcessEnv(&config)
	require.Error(t, err, "conf.ProcessEnv is expected to fail")
	assert.Contains(t, err.Error(), "env: is required but empty for (Value)")
}

func TestProcessEnv_NoDefaultRequired(t *testing.T) {
	type MyConfig struct {
		Foo string `conf:"env:MY_FOO, required"`
	}

	var config MyConfig
	err := conf.ProcessEnv(&config)
	require.Error(t, err, "conf.ProcessEnv is expected to fail")
	assert.Contains(t, err.Error(), "required key (Foo,MY_FOO) missing value")
}

func TestProcessEnv_ProcessFieldFailure(t *testing.T) {
	type MyConfig struct {
		Foo string `conf:"env:MY_FOO"`
		Nbr int    `conf:"env:MY_NBR"`
	}

	setenv(t, "MY_NBR", "abc")

	var config MyConfig
	err := conf.ProcessEnv(&config)
	require.Error(t, err, "conf.ProcessEnv is expected to fail")
	assert.Contains(t, err.Error(), "ProcessField failed (Nbr)")
}

func TestProcessEnvNoPrefix_Success(t *testing.T) {
	os.Clearenv()

	var config SomeFeatureConfig

	setenv(t, "CURRICULUM_DB_CLI_HOST", "localhost")
	setenv(t, "CURRICULUM_DB_HOST", "some-host")
	setenv(t, "CURRICULUM_DB_NAME", "some-name")
	setenv(t, "CURRICULUM_DB_USER", "some-user")
	setenv(t, "CURRICULUM_DB_PASS", "some-pass")
	setenv(t, "APP_NAME", "some-app-name")

	err := conf.ProcessEnv(&config)
	require.NoError(t, err, "conf.ProcessEnv is not expected to fail")
	assert.Equal(t, "some-app-name", config.AppName)
	assert.Equal(t, "foo", config.FeatureField)
	assert.Equal(t, 0, config.IgnoreField)
	assert.Equal(t, "localhost", config.CLIHost)
	assert.Equal(t, "some-host", config.Host)
	assert.Equal(t, "some-name", config.DB.Name)
	assert.Equal(t, "some-user", config.DB.User)
	assert.Equal(t, "some-pass", config.DB.Pass)
	assert.Equal(t, 5432, config.DB.Port)
	assert.True(t, config.DB.IsDebug)
	assert.Equal(t, "/curriculum.log", config.DB.Logfile)
}

func TestEnvVar_Success(t *testing.T) {
	os.Clearenv()
	setenv(t, "FOO", "Bar")

	value, err := conf.EnvVar("FOO")
	require.NoError(t, err)
	assert.Equal(t, "Bar", value)
	os.Clearenv()
}

func TestEnvVar_FailureNotSet(t *testing.T) {
	os.Clearenv()

	_, err := conf.EnvVar("FOO")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "env var (FOO) is not set")
	os.Clearenv()
}

func TestEnvVarStrict_Success(t *testing.T) {
	os.Clearenv()
	setenv(t, "FOO", "Bar")

	value, err := conf.EnvVarStrict("FOO")
	require.NoError(t, err)
	assert.Equal(t, "Bar", value)
	os.Clearenv()
}

func TestEnvVarStrict_FailureNotSet(t *testing.T) {
	os.Clearenv()

	_, err := conf.EnvVarStrict("FOO")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "env var (FOO) is not set")
	os.Clearenv()
}

func TestEnvVarStrict_FailureEmpty(t *testing.T) {
	os.Clearenv()

	setenv(t, "FOO", "")
	_, err := conf.EnvVarStrict("FOO")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "env var (FOO) is empty")
	os.Clearenv()
}

func TestEnvVarOptional(t *testing.T) {
	os.Clearenv()
	setenv(t, "FOO", "Bar")

	value := conf.EnvVarOptional("FOO")
	assert.Equal(t, "Bar", value)

	os.Clearenv()
	value = conf.EnvVarOptional("FOO")
	assert.Equal(t, "", value)

	setenv(t, "FOO", "")
	value = conf.EnvVarOptional("FOO")
	assert.Equal(t, "", value)

}

func setenv(t *testing.T, key, value string) {
	require.NoError(t, os.Setenv(key, value))
}

func TestBindCLI_FieldsFailure(t *testing.T) {
	var config InvalidConfigTagParse

	var cmd = &cobra.Command{
		Use: "my-cmd",
	}

	v := viper.GetViper()

	err := conf.BindCLI(cmd, v, &config)
	require.Error(t, err, "conf.ProcessEnv is expected to fail")
	assert.Contains(t, err.Error(), "Fields failed: parseTag failed (Value)")
}

func TestBindCLI_ParseBoolFailure(t *testing.T) {
	type MyConfig struct {
		ValueB bool `conf:"cli:value-b,cli-u:other usage,default:xya"`
	}

	var config MyConfig

	var cmd = &cobra.Command{
		Use: "my-cmd",
	}

	v := viper.GetViper()

	err := conf.BindCLI(cmd, v, &config)
	require.Error(t, err, "conf.ProcessEnv is expected to fail")
	assert.Contains(t, err.Error(), "strconv.ParseBool failed")
}

func TestBindCLI_Success(t *testing.T) {
	var cmd = &cobra.Command{
		Use: "my-cmd",
	}

	type MyConfig struct {
		ValueA string `conf:"cli:value-a,cli-s:i,cli-u:my usage text,default:abc"`
		ValueB string `conf:"cli:value-b,cli-u:other usage"`
	}

	var config MyConfig
	v := viper.GetViper()

	err := conf.BindCLI(cmd, v, &config)
	require.NoError(t, err, "conf.BindCLI is not expected to failed")

	result := cmd.Flags().Lookup("value-a")
	require.NotNil(t, result, "expecting value-a to be found")
	assert.Equal(t, "value-a", result.Name)
	assert.Equal(t, "i", result.Shorthand)
	assert.Equal(t, "my usage text", result.Usage)
	assert.Equal(t, "abc", result.DefValue)

	result = cmd.Flags().Lookup("value-b")
	require.NotNil(t, result, "expecting value-a to be found")
	assert.Equal(t, "value-b", result.Name)
	assert.Equal(t, "", result.Shorthand)
	assert.Equal(t, "other usage", result.Usage)
	assert.Equal(t, "", result.DefValue)
}

func TestBindCLI_PersistentFlagsSuccess(t *testing.T) {
	var cmd = &cobra.Command{
		Use: "my-cmd",
	}

	type MyConfig struct {
		ValueA string `conf:"cli:value-a,cli-s:i,cli-u:my usage text,default:abc,global-flag"`
		ValueB bool   `conf:"cli:value-b,cli-s:b,cli-u:other usage,default:true,global-flag"`
	}

	var config MyConfig
	v := viper.GetViper()

	err := conf.BindCLI(cmd, v, &config)
	require.NoError(t, err, "conf.BindCLI is not expected to failed")

	result := cmd.PersistentFlags().Lookup("value-a")
	require.NotNil(t, result, "expecting value-a to be found")
	assert.Equal(t, "value-a", result.Name)
	assert.Equal(t, "i", result.Shorthand)
	assert.Equal(t, "my usage text", result.Usage)
	assert.Equal(t, "abc", result.DefValue)

	result = cmd.PersistentFlags().Lookup("value-b")
	require.NotNil(t, result, "expecting value-a to be found")
	assert.Equal(t, "value-b", result.Name)
	assert.Equal(t, "b", result.Shorthand)
	assert.Equal(t, "other usage", result.Usage)
	assert.Equal(t, "true", result.DefValue)
}

func TestBindCLI_WithBoolFieldNoShort_Success(t *testing.T) {
	var cmd = &cobra.Command{
		Use: "my-cmd",
	}

	type MyConfig struct {
		ValueA string `conf:"cli:value-a,cli-s:i,cli-u:my usage text,default:abc"`
		ValueB bool   `conf:"cli:value-b,cli-u:other usage"`
	}

	var config MyConfig
	v := viper.GetViper()

	err := conf.BindCLI(cmd, v, &config)
	require.NoError(t, err, "conf.BindCLI is not expected to failed")

	result := cmd.Flags().Lookup("value-a")
	require.NotNil(t, result, "expecting value-a to be found")
	assert.Equal(t, "value-a", result.Name)
	assert.Equal(t, "i", result.Shorthand)
	assert.Equal(t, "my usage text", result.Usage)
	assert.Equal(t, "abc", result.DefValue)

	result = cmd.Flags().Lookup("value-b")
	require.NotNil(t, result, "expecting value-b to be found")
	assert.Equal(t, "value-b", result.Name)
	assert.Equal(t, "", result.Shorthand)
	assert.Equal(t, "other usage", result.Usage)
	assert.Equal(t, "false", result.DefValue)
}

func TestBindCLI_WithBoolFieldShort_Success(t *testing.T) {
	var cmd = &cobra.Command{
		Use: "my-cmd",
	}

	type MyConfig struct {
		ValueB bool `conf:"cli:value-b,cli-s:b,cli-u:other usage,default:true"`
	}

	var config MyConfig
	v := viper.GetViper()

	err := conf.BindCLI(cmd, v, &config)
	require.NoError(t, err, "conf.BindCLI is not expected to failed")

	result := cmd.Flags().Lookup("value-b")
	require.NotNil(t, result, "expecting value-b to be found")
	assert.Equal(t, "value-b", result.Name)
	assert.Equal(t, "b", result.Shorthand)
	assert.Equal(t, "other usage", result.Usage)
	assert.Equal(t, "true", result.DefValue)
}

func TestBindCLI_SkipCliSuccess(t *testing.T) {
	var cmd = &cobra.Command{
		Use: "my-cmd",
	}

	type MyConfig struct {
		ValueA string `conf:"env:value-a,default:abc"`
	}

	var config MyConfig
	v := viper.GetViper()

	err := conf.BindCLI(cmd, v, &config)
	require.NoError(t, err, "conf.BindCLI is not expected to failed")

	result := cmd.Flags().Lookup("value-a")
	require.Nil(t, result, "expecting value-a to be missing")
}

func TestProcessCLI_SimpleFieldSuccess(t *testing.T) {
	type MyConfig struct {
		Field string `conf:"env:MY_FIELD,default:abc,cmds:my-field,cmds-s:f,cmds-u:some field usage"`
	}

	expectedValue := "foobar"
	cmd := &cobra.Command{
		Use: "my-cmd",
		RunE: func(_ *cobra.Command, args []string) error {
			v := viper.GetViper()
			var config MyConfig

			err := conf.ProcessCLI(v, &config)
			require.NoError(t, err, "conf.ProcessCLI is not expected to fail")
			assert.Equal(t, expectedValue, config.Field)
			return nil
		},
	}

	v := viper.GetViper()
	var config MyConfig

	err := conf.BindCLI(cmd, v, &config)
	require.NoError(t, err, "conf.ProcessCLI is not expected to fail")

	cmd.SetArgs([]string{"--my-field", expectedValue})
	err = cmd.Execute()
}

func TestProcessCLI_SimpleFieldDefaultValue(t *testing.T) {
	type MyConfig struct {
		Field string `conf:"env:MY_FIELD,default:abc,cmds:my-field,cmds-s:f,cmds-u:some field usage"`
	}

	expectedValue := "abc"
	cmd := &cobra.Command{
		Use: "my-cmd",
		RunE: func(_ *cobra.Command, args []string) error {
			v := viper.GetViper()
			var config MyConfig

			err := conf.ProcessCLI(v, &config)
			require.NoError(t, err, "conf.ProcessCLI is not expected to fail")
			assert.Equal(t, expectedValue, config.Field)
			return nil
		},
	}

	v := viper.GetViper()
	var config MyConfig

	err := conf.BindCLI(cmd, v, &config)
	require.NoError(t, err, "conf.ProcessCLI is not expected to fail")

	err = cmd.Execute()
}

func TestProcessCLI_SimpleFieldENVDefaultValue(t *testing.T) {
	type MyConfig struct {
		Field string `conf:"env:MY_FIELD,default:abc"`
	}

	expectedValue := "abc"
	cmd := &cobra.Command{
		Use: "my-cmd",
		RunE: func(_ *cobra.Command, args []string) error {
			v := viper.GetViper()
			var config MyConfig

			err := conf.ProcessCLI(v, &config)
			require.NoError(t, err, "conf.ProcessCLI is not expected to fail")
			assert.Equal(t, expectedValue, config.Field)
			return nil
		},
	}

	v := viper.GetViper()
	var config MyConfig

	err := conf.BindCLI(cmd, v, &config)
	require.NoError(t, err, "conf.ProcessCLI is not expected to fail")

	err = cmd.Execute()
}

func TestProcessCLI_SimpleFieldsSuccess(t *testing.T) {
	type MyConfig struct {
		Field int `conf:"env:MY_FIELD,cli:my-field,cli-s:y,cli-u:some usage"`
	}

	expectedValue := 999
	cmd := &cobra.Command{
		Use: "my-cmd",
		RunE: func(_ *cobra.Command, args []string) error {
			v := viper.GetViper()
			var config MyConfig

			err := conf.ProcessCLI(v, &config)
			require.NoError(t, err, "conf.ProcessCLI is not expected to fail")
			assert.Equal(t, expectedValue, config.Field)
			return nil
		},
	}

	var config MyConfig

	cmd.SetArgs([]string{"--my-field", "999"})
	v := viper.GetViper()
	err := conf.BindCLI(cmd, v, &config)
	require.NoError(t, err, "conf.ProcessCLI is not expected to fail")

	err = cmd.Execute()
	require.NoError(t, err, "cmd.Execute is not expected to fail")
}

func TestProcessCLI_FieldsFailure(t *testing.T) {
	var config InvalidConfigTagParse

	v := viper.GetViper()
	err := conf.ProcessCLI(v, &config)
	require.Error(t, err, "conf.ProcessCLI is expected to fail")
	assert.Contains(t, err.Error(), "Fields failed: parseTag failed (Value)")
}

func TestProcessCLI_RequiredFieldFailure(t *testing.T) {
	type MyConfig struct {
		Field int `conf:"env:MY_FIELD,cli:my-field,cli-s:v,cli-u:some usage,required"`
	}

	cmd := &cobra.Command{
		Use: "my-cmd",
		RunE: func(_ *cobra.Command, args []string) error {
			v := viper.GetViper()
			var config MyConfig

			err := conf.ProcessCLI(v, &config)
			require.Error(t, err, "conf.ProcessCLI is expected to fail")
			assert.Contains(t, err.Error(), "required key (field:Field,env:MY_FIELD,cmds:my-field) missing value")
			return nil
		},
	}

	v := viper.GetViper()
	var config MyConfig

	err := conf.BindCLI(cmd, v, &config)
	require.NoError(t, err, "conf.ProcessCLI is not expected to fail")

	err = cmd.Execute()
	require.NoError(t, err, "cmd.Execute is not expected to fail")
}

func TestProcessCLI_NoValueNoDefaultNotRequired(t *testing.T) {
	type MyConfig struct {
		Field int `conf:"env:MY_FIELD,cmds:my-field,cmds-s:v,cmds-u:some usage"`
	}

	cmd := &cobra.Command{
		Use: "my-cmd",
		RunE: func(_ *cobra.Command, args []string) error {
			v := viper.GetViper()
			var config MyConfig

			err := conf.ProcessCLI(v, &config)
			require.NoError(t, err, "conf.ProcessCLI is not expected to fail")
			assert.Equal(t, 0, config.Field)
			return nil
		},
	}

	v := viper.GetViper()
	var config MyConfig

	err := conf.BindCLI(cmd, v, &config)
	require.NoError(t, err, "conf.BindCLI is not expected to fail")

	err = cmd.Execute()
	require.NoError(t, err, "cmd.Execute is not expected to fail")
}

func TestProcessCLI_ProcessFieldFailure(t *testing.T) {
	type MyConfig struct {
		Field int `conf:"env:MY_FIELD,cmds:my-field,cmds-s:v,cmds-u:some usage"`
	}

	cmd := &cobra.Command{
		Use: "my-cmd",
		RunE: func(_ *cobra.Command, args []string) error {
			v := viper.GetViper()
			var config MyConfig

			err := conf.ProcessCLI(v, &config)
			require.Error(t, err, "conf.ProcessCLI is expected to fail")
			assert.Contains(t, err.Error(), "ProcessField failed (Field)")
			return nil
		},
	}

	setenv(t, "MY_FIELD", "abc")
	v := viper.GetViper()
	var config MyConfig

	err := conf.BindCLI(cmd, v, &config)
	require.NoError(t, err, "conf.BindCLI is not expected to fail")

	err = cmd.Execute()
	require.NoError(t, err, "cmd.Execute is not expected to fail")
}

func TestEnvNames_FieldsFailure(t *testing.T) {
	var config InvalidConfigTagParse

	_, err := conf.EnvNames(&config)
	require.Error(t, err, "conf.EnvNames is expected to fail")
	assert.Contains(t, err.Error(), "Fields failed: parseTag failed (Value)")
}

func TestEnvNames_SuccessWithPrefix(t *testing.T) {
	type MyConfig struct {
		FieldA string `conf:"env:FIELD_A"`
		FieldB string `conf:"env:FIELD_B"`
		FieldC string `conf:"-"`
		FieldD int    `conf:"env:FIELD_D"`
	}

	var config MyConfig
	names, err := conf.EnvNames(&config, "FOO")
	require.NoError(t, err, "conf.EnvNames is not expected to fail")

	expected := []string{"FOO_FIELD_A", "FOO_FIELD_B", "FOO_FIELD_D"}
	assert.Equal(t, expected, names)
}

func TestEnvNames_SuccessWithNoPrefix(t *testing.T) {
	type MyConfig struct {
		FieldA string `conf:"env:FIELD_A"`
		FieldB string `conf:"env:FIELD_B"`
		FieldC string `conf:"-"`
		FieldD int    `conf:"env:FIELD_D"`
	}

	var config MyConfig
	names, err := conf.EnvNames(&config)
	require.NoError(t, err, "conf.EnvNames is not expected to fail")

	expected := []string{"FIELD_A", "FIELD_B", "FIELD_D"}
	assert.Equal(t, expected, names)
}

func TestEnvToMap_FieldsFailure(t *testing.T) {
	var config InvalidConfigTagParse

	_, err := conf.EnvToMap(&config)
	require.Error(t, err, "conf.EnvToMap is expected to fail")
	assert.Contains(t, err.Error(), "Fields failed: parseTag failed (Value)")
}

func TestEnvToMap_Success(t *testing.T) {
	type MyConfig struct {
		FieldA string `conf:"env:FIELD_A,default:abc"`
		FieldB string `conf:"env:FIELD_B,required"`
		FieldC string `conf:"-"`
		FieldD int    `conf:"env:FIELD_D,default:888"`
		FieldE int    `conf:"env:FIELD_E"`
		FieldX string `conf:"env:-,cmds:abc"`
	}

	setenv(t, "FIELD_B", "xyz")
	setenv(t, "FIELD_D", "777")

	var config MyConfig
	result, err := conf.EnvToMap(&config)
	require.NoError(t, err, "conf.EnvNames is not expected to fail")

	expected := map[string]string{
		"FIELD_A": "abc",
		"FIELD_B": "xyz",
		"FIELD_D": "777",
		"FIELD_E": "",
	}
	assert.Equal(t, expected, result)
}

func TestEnvToMap_EnvMissingFailure(t *testing.T) {
	type MyConfig struct {
		FieldA string `conf:"cmds:abc, default:abc"`
		FieldB string `conf:"env:FIELD_B,required"`
		FieldC string `conf:"-"`
		FieldD int    `conf:"env:FIELD_D,default:888"`
		FieldX string `conf:"env:-,cmds:abc"`
	}

	setenv(t, "FIELD_B", "xyz")
	setenv(t, "FIELD_D", "777")

	var config MyConfig
	_, err := conf.EnvToMap(&config)
	require.Error(t, err, "conf.EnvToMap is expected to fail")

	assert.Contains(t, err.Error(), "env: is required but empty for (FieldA)")
}

func TestEnvToMap_RequiredFailure(t *testing.T) {
	type MyConfig struct {
		FieldA string `conf:"env:FIELD_A,default:abc"`
		FieldB string `conf:"env:FIELD_B,required"`
		FieldC string `conf:"-"`
		FieldD int    `conf:"env:FIELD_D,default:888"`
		FieldX string `conf:"env:-,cmds:abc"`
	}

	os.Clearenv()
	setenv(t, "FIELD_D", "777")

	var config MyConfig
	_, err := conf.EnvToMap(&config)
	require.Error(t, err, "conf.EnvToMap is expected to fail")

	assert.Contains(t, err.Error(), "required key (FieldB,FIELD_B) missing value")
}

func TestProcessParamStore_NoAPI_Failure(t *testing.T) {
	type Config struct {
		FieldA string
	}

	var config Config

	_, err := conf.ProcessParamStore(nil, "some_title", false, &config)
	require.Error(t, err, "conf.ProcessParamStore is expected to fail")
	assert.Contains(t, err.Error(), "pstore is nil")
}

//
// func TestProcessParamStore_NoAppTitle_Failure(t *testing.T) {
// 	type Config struct {
// 		FieldA string
// 	}
//
// 	var config Config
//
// 	sess, err := session.NewSession()
// 	require.NoError(t, err)
//
// 	store := conf.PStore{
// 		API: ssm.New(sess),
// 	}
// 	_, err = conf.ProcessParamStore(store, &config)
// 	require.Error(t, err, "conf.ProcessParamStore is expected to fail")
// 	assert.Contains(t, err.Error(), "pstore.AppTitle is empty")
// }
//
// func TestProcessParamStore_Fields_Failure(t *testing.T) {
// 	type Config struct {
// 		FieldA bool `conf:"env:,default:xyz"`
// 	}
//
// 	var config Config
//
// 	sess, err := session.NewSession()
// 	require.NoError(t, err)
//
// 	store := conf.PStore{
// 		API:      ssm.New(sess),
// 		AppTitle: "Foo",
// 	}
//
// 	_, err = conf.ProcessParamStore(store, &config)
// 	require.Error(t, err, "conf.ProcessParamStore is expected to fail")
// 	assert.Contains(t, err.Error(), "Fields failed: parseTag failed (FieldA)")
// }
//
// func TestProcessParamStore_EnvIsEmpty_Failure(t *testing.T) {
// 	type Config struct {
// 		FieldY string `conf:"-"`
// 		FieldZ string `conf:"pstore:-"`
// 		FieldX string `conf:"env:-"`
// 		FieldA bool   `conf:"default:xyz"`
// 	}
//
// 	var config Config
//
// 	sess, err := session.NewSession()
// 	require.NoError(t, err)
//
// 	store := conf.PStore{
// 		API:      ssm.New(sess),
// 		AppTitle: "Foo",
// 	}
//
// 	_, err = conf.ProcessParamStore(store, &config)
// 	require.Error(t, err, "conf.ProcessParamStore is expected to fail")
// 	assert.Contains(t, err.Error(), "env: is required but empty for (FieldA)")
// }
//
// func TestProcessParamStore_APIGetParameter_Failure(t *testing.T) {
// 	type Config struct {
// 		FieldA bool `conf:"env:Foo, default:xyz"`
// 	}
//
// 	var config Config
//
// 	store := conf.PStore{
// 		API: &MockSSM{
// 			GetParamError: errors.New("some api error"),
// 		},
// 		AppTitle: "MyService",
// 	}
//
// 	_, err := conf.ProcessParamStore(store, &config)
// 	require.Error(t, err, "conf.ProcessParamStore is expected to fail")
// 	assert.Contains(t, err.Error(), "pstore.API.GetParameter failed for (FieldA, /MyService/Foo)")
// }
//
// // This should never happen in reality
// func TestProcessParamStore_APIGetParameterReturnsNil_Failure(t *testing.T) {
// 	type Config struct {
// 		FieldA bool `conf:"env:Foo, pstore:STORE_FIELD, default:xyz"`
// 	}
//
// 	var config Config
//
// 	store := conf.PStore{
// 		API:      &MockSSM{},
// 		AppTitle: "MyService",
// 	}
//
// 	_, err := conf.ProcessParamStore(store, &config)
// 	require.Error(t, err, "conf.ProcessParamStore is expected to fail")
// 	assert.Contains(t, err.Error(), "pstore.API.GetParameter returned nil (FieldA, STORE_FIELD)")
// }
//
// func TestProcessParamStore_Required_Failure(t *testing.T) {
// 	type Config struct {
// 		FieldA bool `conf:"env:Foo, pstore:STORE_FIELD, required"`
// 	}
//
// 	var config Config
//
// 	store := conf.PStore{
// 		API: &MockSSM{
// 			GetParamResponse: &ssm.GetParameterOutput{
// 				Parameter: &ssm.Parameter{},
// 			},
// 		},
// 		AppTitle: "MyService",
// 	}
//
// 	_, err := conf.ProcessParamStore(store, &config)
// 	require.Error(t, err, "conf.ProcessParamStore is expected to fail")
// 	assert.Contains(t, err.Error(), "required key (FieldA,STORE_FIELD) missing value")
// }
//
// func TestProcessParamStore_Success(t *testing.T) {
// 	type Config struct {
// 		FieldA bool `conf:"env:Foo, pstore:STORE_FIELD, required"`
// 	}
//
// 	var config Config
//
// 	pstoreValue := "MyValue"
// 	store := conf.PStore{
// 		API: &MockSSM{
// 			GetParamResponse: &ssm.GetParameterOutput{
// 				Parameter: &ssm.Parameter{
// 					Value: aws.String(pstoreValue),
// 				},
// 			},
// 		},
// 		AppTitle: "MyService",
// 	}
//
// 	result, err := conf.ProcessParamStore(store, &config)
// 	require.NoError(t, err, "conf.ProcessParamStore is not expected to fail")
// 	require.NotNil(t, result)
//
// 	expected, ok := result["Foo"]
//
// 	require.True(t, ok)
// 	assert.Equal(t, pstoreValue, expected)
// }
//
// func TestProcessParamStore_DefaultValue_Success(t *testing.T) {
// 	type Config struct {
// 		FieldA bool `conf:"env:Foo, pstore:STORE_FIELD, default:MyDefault"`
// 	}
//
// 	var config Config
//
// 	store := conf.PStore{
// 		API: &MockSSM{
// 			GetParamResponse: &ssm.GetParameterOutput{
// 				Parameter: &ssm.Parameter{},
// 			},
// 		},
// 		AppTitle: "MyService",
// 	}
//
// 	result, err := conf.ProcessParamStore(store, &config)
// 	require.NoError(t, err, "conf.ProcessParamStore is not expected to fail")
// 	require.NotNil(t, result)
//
// 	expected, ok := result["Foo"]
//
// 	require.True(t, ok)
// 	assert.Equal(t, "MyDefault", expected)
// }

type MockSSM struct {
	ssmiface.SSMAPI
	GetParamResponse *ssm.GetParameterOutput
	GetParamError    error
}

func (m *MockSSM) GetParameter(_ *ssm.GetParameterInput) (*ssm.GetParameterOutput, error) {
	return m.GetParamResponse, m.GetParamError
}
