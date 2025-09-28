#!/usr/bin/env perl
package filter_locales;

=head1 NAME

filter_locales.pl - Filter locale files to keep only used localization keys

=head1 SYNOPSIS

    perl filter_locales.pl [OPTIONS]

=head1 DESCRIPTION

This script filters all locale JSON files to keep only the localization keys
that are actually used in the codebase. It reads the list of used keys from
keys-cod.csv and removes all unused keys from the locale files.

Directories & Files:
- CSV:  $PROJECT_ROOT/locales/
- JSON: $PROJECT_ROOT/internal/locales/
- Keys: $PROJECT_ROOT/locales/keys-cod.csv

=head1 OPTIONS

    --csv           Filter CSV files(default)
    --json          Filter JSON files
    --keys          Use specified keys file
    --add_empty     Add empty translation unless it exists in the original
    --help          Show this help message
    --verbose       Enable verbose output

=head1 EXAMPLES

    # Filter CSV
    perl filter_locales.pl

    # Filter JSON
    perl filter_locales.pl --json

    # Filter JSON with verbose output
    perl filter_locales.pl --json --verbose

    # Filter CSV with different keys file
    perl filter_locales.pl --keys /path/to/keys_file.csv

=cut

#*******************************************************************************

use strict;
use warnings 'all';
use utf8;
use v5.38.2;

our $VERSION = "0.1.0";

#*******************************************************************************

use English qw(-no_match_vars);
use Data::Dumper;
use Getopt::Long;
use Pod::Usage;
use JSON;
use Text::CSV;
use File::Spec;
use File::Basename;
use Cwd 'abs_path';
use Const::Fast;

#*******************************************************************************
const my @_OPTIONS => (
    'help|h|?',
    'man',
    'verbose|v',
    'json=s',
    'keys|k=s',
    'add_empty|ae',
);

#*******************************************************************************
sub get_options {

    my $options = {};
    if (!GetOptions($options, @_OPTIONS)) {
        pod2usage(-verbose => 0, -exitval => 2, -output => \*STDERR, -message => "Invalid option(s)");
    }

    if ($options->{help}) {
        pod2usage(-verbose => 0, -exitval => 2, -output => \*STDERR);
    }

    if ($options->{man}) {
        pod2usage(-verbose => 1, -exitval => 2, -output => \*STDERR);
    }

    return $options;
}

#*******************************************************************************
sub prepare {
    # Enable UTF-8 for all I/O
    binmode(STDOUT, ':encoding(UTF-8)');
    binmode(STDERR, ':encoding(UTF-8)');

    my $options = get_options();

    # Find project root
    $options->{script_dir}   = dirname(abs_path($0));
    $options->{project_root} = dirname($options->{script_dir});

    $options->{keys}     = File::Spec->catfile($options->{project_root}, 'locales', 'keys-cod.csv');
    $options->{json_dir} = File::Spec->catdir($options->{project_root}, 'internal', 'locales');
    $options->{csv_dir}  = File::Spec->catdir($options->{project_root}, 'locales');

    print "Project root: $options->{project_root}\n";
    print "JSON directory: $options->{json_dir}\n";
    print "CSV directory: $options->{csv_dir}\n";
    print "Used keys file: $options->{keys}\n";

    return $options;
} ## end sub prepare

#*******************************************************************************
sub read_list_of_keys ($keys_file) {

    # Read the list of used keys
    unless (-f $keys_file) {
        die "Used keys file not found: $keys_file\n";
    }

    my %used_keys;
    open my $keys_fh, '<:encoding(utf8)', $keys_file
        or die "Cannot open used keys file '$keys_file': $!\n";

    while (my $line = <$keys_fh>) {
        chomp $line;
        next if $line eq '' || $line =~ /^\s*$/;    # Skip empty lines
        $used_keys{$line} = 1;
    }
    close $keys_fh;

    my $used_key_count = keys %used_keys;
    print "Loaded $used_key_count used keys\n";

    return \%used_keys;
} ## end sub read_list_of_keys

#*******************************************************************************
sub read_json_file ($file) {
    open my $fh, '<:encoding(utf8)', $file
        or die "Cannot open JSON file '$file': $!";

    my $json_content = do {local $/; <$fh>};
    close $fh;

    my $json = JSON->new();
    return $json->decode($json_content);
}

#*******************************************************************************
sub read_json_file_hash ($file) {
    my $data = read_json_file($file);
    if (ref $data eq 'HASH') {
        return $data;
    }

    die "Invalid JSON format in '$file'. Expected hash object.\n";
}

#*******************************************************************************
sub write_json_file_hash ($data, $file) {
    my $output_json = JSON->new->pretty->canonical;

    open my $fh, '>:encoding(utf8)', $file
        or die "Cannot create JSON file '$file': $!";

    print $fh $output_json->encode($data)
        or die "Cannot write to JSON file '$file': $!";

    close $fh;
}

#*******************************************************************************
sub read_csv_file_hash ($file) {
    my $csv = Text::CSV->new({binary => 1, auto_diag => 1});

    open my $fh, '<:encoding(utf8)', $file
        or die "Cannot open CSV file '$file': $!";

    my $result = {};
    while (my $row = $csv->getline($fh)) {
        next if (($row->[0] eq '') || ($row->[0] =~ /^\s*$/) || ($row->[0] eq 'Key'));    # Skip empty lines
        $result->{$row->[0]} = $row->[1] || '';
    }
    close $fh;

    return $result;
}

#*******************************************************************************
sub write_csv_file_hash ($data, $file) {
    my $csv = Text::CSV->new({binary => 1, auto_diag => 1});

    open my $fh, '>:encoding(utf8)', $file
        or die "Cannot create CSV file '$file': $!";

        $csv->say($fh, ['Key','Value'])
        or die "Cannot write to CSV file '$file': $!";

    for my $key (sort keys %{$data}) {
        $csv->say($fh, [$key, $data->{$key}])
        or die "Cannot write to CSV file '$file': $!";
    }

    close $fh;
}

#*******************************************************************************
sub read_languages ($options) {

    my $languages_file = File::Spec->catfile($options->{json_dir}, "languages.json");

    return read_json_file_hash($languages_file);
}

#*******************************************************************************
sub run {
    my ($self) = @_;

    my $options = prepare();

    my $used_keys = read_list_of_keys($options->{keys});

    my $languages = read_languages($options);

    # Process each locale file
    for my $locale (sort keys %{$languages}) {
        my $file;
        my $translations;

        if ($options->{json}) {
            print "Processing $locale.json...\n";

            $file         = File::Spec->catfile($options->{json_dir}, "$locale.json");
            $translations = read_json_file_hash($file);
        }
        else {
            print "Processing $locale.csv...\n";

            $file         = File::Spec->catfile($options->{csv_dir}, "$locale.csv");
            $translations = read_csv_file_hash($file);
        }

        # ----------------

        # Filter to keep only used keys
        my %filtered_translations;
        my $original_count = keys %$translations;
        my $removed_count  = 0;

        if ($options->{add_empty}) {
            for my $key (keys %{$used_keys}) {
                $filtered_translations{$key} = $translations->{$key} || '';
            }
        }
        else {
            for my $key (keys %{$translations}) {
                if (exists $used_keys->{$key}) {
                    $filtered_translations{$key} = $translations->{$key};
                }
                else {
                    $removed_count++;
                }
            }
        }

        my $kept_count = keys %filtered_translations;

        # Create backup
        my $backup_file = "$file.backup";
        rename($file, $backup_file) or die "Cannot create backup: $!";

        # ----------------

        if ($options->{json}) {
            print "Writing $locale.json...\n";
            write_json_file_hash(\%filtered_translations, $file);
        }
        else {
            print "Writing $locale.csv...\n";
            write_csv_file_hash(\%filtered_translations, $file);
        }

        print "  -> Original: $original_count keys\n";
        print "  -> Kept: $kept_count keys\n";
        print "  -> Removed: $removed_count keys\n" unless ($options->{add_empty});
        print "  -> Backup saved as: $backup_file\n";
    } ## end for my $locale (sort keys...)

    print "\nFiltering complete!\n";
    print "All locale files have been filtered to contain only used keys.\n";
    print "Backup files (.backup) have been created for safety.\n";

    return 1;
} ## end sub run

#*******************************************************************************
__PACKAGE__->run();

#*******************************************************************************
__END__

=head1 AUTHOR

ShoPogoda Bot Development Team

=cut
